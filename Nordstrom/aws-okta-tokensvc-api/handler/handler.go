/*
AWS Okta -- https://gitlab.nordstrom.com/public-cloud/aws-okta
Maintained by Cloud Engineering <cloudengineering@nordstrom.com>
	Author: Jason Casas

Copyright 2020 @ Nordstrom, Inc. All rights reserved.
*/

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/coreos/go-oidc"
	"gitlab.nordstrom.com/public-cloud/aws-okta-token-service/lib"
)

// Handler is our lambda handler invoked by the `lambda.Start` function call
func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var (
		environment string
		contentType string
		stage       string
	)

	logConfig := &lib.LogConfig{
		Format: lib.LogFormat,
		Level:  lib.LogLevel,
	}

	logger, err := lib.NewLogProvidor(logConfig)
	if err != nil {
		return lib.ServerError(req.RequestContext.RequestID, err)
	}

	log := logger

	log.Infof("client_request_id: %s", req.RequestContext.RequestID)

	caller := req.Headers["caller"]

	stg := req.Headers["stage"]
	if stg != "" {
		stage = stg
	} else {
		stage = "prod"
	}

	env := req.Headers["environment"]
	if env != "" {
		environment = env
	} else {
		environment = "nonpci"
	}
	log.Debugf("stage: %s, environment: %s, caller: %s",
		stage,
		environment,
		caller,
	)

	ct := req.Headers["Content-Type"]
	if ct != "" {
		contentType = ct
	} else {
		contentType = req.Headers["content-type"]
	}

	if contentType == "" {
		err := fmt.Errorf("missing content-type")
		return lib.ClientError(req.RequestContext.RequestID, http.StatusNotAcceptable, err)
	}

	if contentType != "application/json" {
		err := fmt.Errorf("Invalid contentType: expected \"application/json\" got \"%s\"", contentType)
		return lib.ClientError(req.RequestContext.RequestID, http.StatusNotAcceptable, err)
	}

	cfgOps := &lib.ProvidorOptions{
		Stage:       stage,
		Environment: environment,
		LogLevel:    lib.LogLevel,
	}

	cfg, err := lib.NewBaseConfig(*cfgOps)
	if err != nil {
		return lib.ServerError(req.RequestContext.RequestID, err)
	}

	log.Debugf("OIDC parameters: (%+v)", cfg.OIDC)

	// Create OIDC provider
	p, err := oidc.NewProvider(ctx, cfg.OIDC.OktaServerUS.String())
	if err != nil {
		return lib.ServerError(req.RequestContext.RequestID, err)
	}

	provider := p

	oidcConfig := &oidc.Config{
		ClientID: cfg.OIDC.ClientID,
	}

	// Verifier provides verification for ID Tokens
	verifier := provider.Verifier(oidcConfig)

	// Parse and verify ID Token payload
	log.Info("Verifying ID Token")

	rawIDToken := req.Headers["Authorization"]
	if rawIDToken == "" {
		err := fmt.Errorf("missing id-token")
		return lib.ClientError(req.RequestContext.RequestID, http.StatusNotAcceptable, err)
	}

	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		log.Errorf("Visit: (%s) and paste %q to review the client provided ID-Token", "http://https://jwt.io/", rawIDToken)
		return lib.ClientError(req.RequestContext.RequestID, http.StatusUnauthorized, fmt.Errorf("RequestId: [%s]: (%+v)", req.RequestContext.RequestID, err))
	}

	// Extract the Groups Custom Claims
	// Sub is subject of the token (unique user identifer), e.g., "00umqxus8TROl4PPS2p6"
	// Groups is user's AWS AD Security Groups, e.g., "AWS#NORD-SANDBOXTEAM01-DevUsers-Team#468543742856"
	var claims struct {
		Email string `json:"email"`
		Sub   string `json:"sub"`
	}

	if err := idToken.Claims(&claims); err != nil {
		return lib.ServerError(req.RequestContext.RequestID, err)
	}

	awsGroups, err := lib.GetADGroups(claims.Email, cfg.AskLDAP)
	if err != nil {
		return lib.ServerError(req.RequestContext.RequestID, err)
	}

	envGroups, err := lib.GetEnvGroups(cfg.Environment, awsGroups)
	if err != nil {
		return lib.ServerError(req.RequestContext.RequestID, err)
	}

	groupArns := lib.GroupsToARNs(envGroups)

	// here we branch off - if caller is groups provider, we return a list of groups
	if caller == "groups" {
		log.Infof("%s requested: groups", claims.Email)

		sort.Strings(groupArns)

		var buf bytes.Buffer
		body, err := json.Marshal(map[string]interface{}{
			"Groups": groupArns,
		})
		if err != nil {
			return lib.ServerError(req.RequestContext.RequestID, err)
		}
		json.HTMLEscape(&buf, body)

		log.Infof("client_request_id: %s, status: %s, response: %s", req.RequestContext.RequestID, "ok", "returning role arns")
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusOK,
			Body:       buf.String(),
		}, nil
	}

	type role struct {
		RoleArn         string `json:"RoleArn"`
		RoleSessionName string `json:"RoleSessionName"`
		DurationSeconds int64  `json:"DurationSeconds"`
		Policy          string `json:"Policy"`
	}

	r := &role{}

	err = json.Unmarshal([]byte(req.Body), r)
	if err != nil {
		return lib.ClientError(req.RequestContext.RequestID, http.StatusNotAcceptable, err)
	}

	// to make RoleSessionName unique we prefix the Okta Sub unique identitifer
	r.RoleSessionName = lib.UniqueSessionName(claims.Sub, claims.Email)

	log.Infof("%s requested: assume-role (%s)",
		claims.Email,
		strings.Split(r.RoleArn, ":")[5][5:],
	)

	// RoleArn cannot be left empty
	if r.RoleArn == "" {
		err := fmt.Errorf("missing role_arn")
		return lib.ClientError(req.RequestContext.RequestID, http.StatusNotAcceptable, err)
	}

	// Validate if RoleArn to assume is AWS ARN format
	if !(lib.ValidRole.MatchString(r.RoleArn)) {
		err := fmt.Errorf("invalid role-arn")
		return lib.ClientError(req.RequestContext.RequestID, http.StatusNotAcceptable, err)
	}

	// Check if RoleArn to assume is in list of User's RoleArns
	if !(lib.Contains(groupArns, r.RoleArn)) {
		err := fmt.Errorf("(%s) User: %s cannot assume role: %s",
			cfg.Environment,
			claims.Email,
			r.RoleArn,
		)
		return lib.ClientError(req.RequestContext.RequestID, http.StatusUnauthorized, err)
	}

	// AssumeRole Input: https://amzn.to/2IHqbaM
	// AssumeRole Output: https://amzn.to/2EaglPQ
	sess := session.Must(session.NewSession())
	svc := sts.New(sess)

	stsParams := &sts.AssumeRoleInput{}

	if r.Policy != "" {
		stsParams = &sts.AssumeRoleInput{
			RoleArn:         aws.String(r.RoleArn),
			RoleSessionName: aws.String(r.RoleSessionName),
			DurationSeconds: aws.Int64(int64(r.DurationSeconds)),
			Policy:          aws.String(r.Policy),
		}
	} else {
		stsParams = &sts.AssumeRoleInput{
			RoleArn:         aws.String(r.RoleArn),
			RoleSessionName: aws.String(r.RoleSessionName),
			DurationSeconds: aws.Int64(int64(r.DurationSeconds)),
		}
	}

	log.Debugf("assume-role parameters: %+v",
		stsParams,
	)

	assumeResp, err := svc.AssumeRole(stsParams)
	if err != nil {
		return lib.ServerError(req.RequestContext.RequestID, err)
	}

	var buf bytes.Buffer
	body, err := json.Marshal(map[string]interface{}{
		"Credentials":     assumeResp.Credentials,
		"AssumedRoleUser": assumeResp.AssumedRoleUser,
	})

	if err != nil {
		return lib.ServerError(req.RequestContext.RequestID, err)
	}

	json.HTMLEscape(&buf, body)

	log.Infof("client_request_id: %s, status: %s, response: %s", req.RequestContext.RequestID, "ok", "returning credentials")
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       buf.String(),
	}, nil
}
