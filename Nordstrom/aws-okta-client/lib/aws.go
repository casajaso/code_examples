/*
AWS Okta -- https://gitlab.nordstrom.com/public-cloud/aws-okta
Maintained by Cloud Engineering <cloudengineering@nordstrom.com>
	Author: Jason Casas

Copyright 2020 @ Nordstrom, Inc. All rights reserved.
*/

package lib

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/99designs/keyring"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/manifoldco/promptui"
	"gopkg.in/ini.v1"
)

// DefaultRoleSessionName is used if no role_session_name is configured for profile
// TokenService is the host of Token Service
var (
	DefaultRoleSessionName = GetEnv("AWS_OKTA_DEFAULT_ROLE_SESSION_NAME", AWSSessionName)
)

// AWSProvider Provider data
type AWSProvider struct {
	credentials.Expiry
	profile     string
	duration    time.Duration
	keyring     keyring.Keyring
	session     *AWSKeyringSession
	profiles    Profiles
	sessionName string
	Config      *Config
}

type Profiles map[string]map[string]string

// awsSession structures data for STS credentials and session name
type awsSession struct {
	sts.Credentials
	Name string
}

// AWSKeyringSessions structures data for a Keyring and profiles
type AWSKeyringSession struct {
	Keyring  keyring.Keyring
	Profiles Profiles
}

// AWSCredsProvider inits Profile Provider
func AWSCredsProvider(o ProvidorOptions) (*AWSProvider, error) {
	if err := checkDuration(o.AssumeRoleDuration, o.Config.AWS.SessionMin, o.Config.AWS.SessionMax); err != nil {
		return nil, err
	}

	// sessionName := sessionKey(AWSSessionName, o.ProfileName, o.AssumeRoleDuration)
	sessionName := fmt.Sprintf("%s,%s", AWSSessionName, o.ProfileName)

	krSession := &AWSKeyringSession{o.Keyring, o.Profiles}

	return &AWSProvider{
		keyring:     o.Keyring,
		session:     krSession,
		profile:     o.ProfileName,
		profiles:    krSession.Profiles,
		Config:      o.Config,
		duration:    o.AssumeRoleDuration,
		sessionName: sessionName,
	}, nil
}

// Validate - ensures requested duration is within limits
func checkDuration(duration time.Duration, min time.Duration, max time.Duration) error {
	if duration < min {
		return fmt.Errorf("Minimum assume-role duration is: (%s) duration requested: (%s) ", min.String(), duration)
	} else if duration > max {
		return fmt.Errorf("Maximum assume-role duration is: (%s) duration requested: (%s) ", max.String(), duration)
	}

	return nil
}

// Retrieve assumes AWS IAM role from AWS Okta Token Service and returns temporary credentials
func (a *AWSProvider) Retrieve(prf *ini.Section) (credentials.Value, time.Time, error) {

	sessionToken, _, err := a.session.GetSessionToken(a.sessionName)

	if err == nil {
		var resume bool

		if a.Config.Environment != "pci" && time.Until(*sessionToken.Expiration) >= 5 {
			resume = true
		}

		if a.Config.Environment == "pci" && time.Until(*sessionToken.Expiration) >= 1 {
			resume = true
		}

		if resume {
			log.Infof("AWS Session Token expiry: (%s) - resuming session",
				sessionToken.Expiration.Sub(time.Now()).String())

			value := credentials.Value{
				AccessKeyID:     *sessionToken.AccessKeyId,
				SecretAccessKey: *sessionToken.SecretAccessKey,
				SessionToken:    *sessionToken.SessionToken,
				ProviderName:    "okta",
			}

			return value, *sessionToken.Expiration, nil
		}
	}

	profileARN, err := prf.GetKey("_role_arn")
	if err != nil {
		return credentials.Value{}, time.Time{}, err
	}

	log.Infof("Requesting AWS Session credentials for: (%s)",
		strings.Split(profileARN.String(), ":")[5][5:],
	)

	o := &OktaSessionToken{}

	item, err := a.keyring.Get(fmt.Sprintf("%s,%s,%s", OktaSessionName, a.Config.Stage, a.Config.Environment))
	if err != nil {
		return credentials.Value{}, time.Time{}, fmt.Errorf("Unable to retrieve %s: (%s)", OktaSessionName, err)
	}

	if err = json.Unmarshal(item.Data, &o); err != nil {
		return credentials.Value{}, time.Time{}, err
	}

	//Okta session token refresh logic
	if !o.SessionToken.Valid() {
		log.Infof("Okta Session for (%s) expired - requesting new token", a.Config.Environment)

		err := RefreshOktaSessionToken(a.keyring, item, a.Config)
		if err != nil {
			return credentials.Value{}, time.Time{}, err
		}

		item, err := a.keyring.Get(fmt.Sprintf("%s,%s,%s", OktaSessionName, a.Config.Stage, a.Config.Environment))
		if err != nil {
			return credentials.Value{}, time.Time{}, err
		}

		if err = json.Unmarshal(item.Data, &o); err != nil {
			return credentials.Value{}, time.Time{}, err
		}
	}

	// creds, err := a.assumeRoleFromSession(o.IDToken, profileARN.String(), a.duration)
	creds, err := a.assumeRoleFromSession(o.IDToken, profileARN.String(), a.duration)
	if err != nil {
		return credentials.Value{}, time.Time{}, err
	}

	a.SetExpiration(*creds.Expiration, a.duration)

	log.Infof("Granted AWS session token expiring in: (%s)",
		creds.Expiration.Sub(time.Now()).String(),
	)

	// PutAWSSessionKey Store New Session in Keyring
	a.session.SaveSessionToken(a.profile, a.sessionName, creds)

	value := credentials.Value{
		AccessKeyID:     *creds.AccessKeyId,
		SecretAccessKey: *creds.SecretAccessKey,
		SessionToken:    *creds.SessionToken,
		ProviderName:    "okta",
	}

	//return value, *session.Expiration, nil
	return value, *creds.Expiration, nil

}

func (a *AWSProvider) GetTokenInfo() (sts.Credentials, string, error) {
	log.Infof("Retrieving: (%s)", a.sessionName)

	sessionToken, _, err := a.session.GetSessionToken(a.sessionName)

	if err != nil {
		return sts.Credentials{}, "", err
	}

	expire := sessionToken.Expiration.Sub(time.Now()).String()

	return sessionToken, expire, nil
}

// assumeRoleFromSession takes a session created with an okta login IdP token and uses that to assume a role
func (a *AWSProvider) assumeRoleFromSession(rawIDToken string, profileARN string, duration time.Duration) (sts.Credentials, error) {
	//func (a *AWSProvider) assumeRoleFromSession(rawIDToken string, profileARN string, duration time.Duration, policy string) (sts.Credentials, error) {
	authAuthenticatorURL := a.Config.OIDC.TokenSVC.String()

	name := a.roleSessionName()

	data := map[string]interface{}{
		"RoleArn":         profileARN,
		"RoleSessionName": name,
		"DurationSeconds": int64(duration.Seconds()),
		//"Policy": policy //PH - JCC
	}

	bytesRepresentation, err := json.Marshal(data)
	if err != nil {
		return sts.Credentials{}, err
	}

	client := &http.Client{}

	req, err := http.NewRequest("POST", authAuthenticatorURL, bytes.NewBuffer(bytesRepresentation))
	if err != nil {
		return sts.Credentials{}, err
	}
	req.Header.Add("Authorization", rawIDToken)
	req.Header.Add("Stage", a.Config.Stage)
	req.Header.Add("Environment", a.Config.Environment)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("Caller", "authenticate")

	resp, err := client.Do(req)
	if err != nil {
		return sts.Credentials{}, err
	}

	log.Infof("Requesting authorization - Client-RequestId: [%s]", resp.Header.Get("x-amzn-RequestId"))

	if resp == nil {
		err := fmt.Errorf("(%s) returned empty response", authAuthenticatorURL)
		return sts.Credentials{}, err
	}

	defer resp.Body.Close()

	var result map[string]interface{}

	json.NewDecoder(resp.Body).Decode(&result)

	if resp.StatusCode >= 400 {
		return sts.Credentials{}, fmt.Errorf(result["message"].(string))
	}

	b, err := json.Marshal(result["Credentials"])
	if err != nil {
		return sts.Credentials{}, err
	}

	creds := sts.Credentials{}

	err = json.Unmarshal(b, &creds)
	if err != nil {
		return sts.Credentials{}, err
	}

	return creds, nil
}

// DeleteAWSSessions removes current AWS session from keyring
func (a *AWSProvider) DeleteAWSSession() error {

	keys, err := a.keyring.Keys()
	if err != nil {
		return fmt.Errorf("Unable to search keyring keys")
	}

	for _, k := range keys {
		if strings.Contains(k, AWSSessionName) {
			prof := strings.Split(k, ",")[1]
			if prof == a.profile {
				log.Infof("Removing AWS Session token: (%s)", k)
				err := a.keyring.Remove(k)
				if err != nil {
					return err
				}
			}

		}
	}
	return nil
}

// DeleteAWSSessions removes all AWS session keys from keyring
func DeleteAWSSessions(kr keyring.Keyring) error {

	keys, err := kr.Keys()
	if err != nil {
		return fmt.Errorf("Unable to search keyring keys")
	}

	log.Debugf("Searching keyring for: (%s) Items", AWSSessionName)

	for _, k := range keys {
		if strings.Contains(k, AWSSessionName) {
			log.Infof("Found session: (%s)", strings.Split(k, ","))
			log.Info("Terminating..")
			err := kr.Remove(k)
			if err != nil {
				return fmt.Errorf("Unable to remove session key: %s (%s)", k, err)
			}
		}

	}
	return nil
}

func SelectRole(role string, groups []string) (string, error) {
	var r bool
	if role != "" {
		ok := Contains(groups, role)
		if !ok {
			r = false
			return "", fmt.Errorf("Invalid role_arn (%s): (malformed or not-authorized)", role)
		} else {
			r = true
		}
	}

	if r != true {
		prompt := promptui.Select{
			Label: "Select AWS Federated Role",
			Items: groups,
		}

		_, result, err := prompt.Run()
		if err != nil {
			return "", err

		} else {
			return result, nil
		}

	} else {
		return role, nil
	}
}

// AWSSessionKey formats AWS sessionfor retrieval in the Keyring
func sessionKey(sessionName string, profile string, duration time.Duration) string {

	hasher := md5.New()

	hasher.Write([]byte(duration.String()))

	enc := json.NewEncoder(hasher)

	enc.Encode(profile)

	sessionKey := fmt.Sprintf("%s %s (%x)", sessionName, profile, hex.EncodeToString(hasher.Sum(nil))[0:10])

	if sessionKey != "" {
		fmt.Sprintln(fmt.Errorf("Failed to define AWS Session Key"))
	}

	return sessionKey
}

// Retrieve - retieves a profile session / keyring item from the Keyring
func (s *AWSKeyringSession) GetSessionToken(sessionName string) (sts.Credentials, string, error) {
	var session awsSession

	item, err := s.Keyring.Get(sessionName)
	if err != nil {
		return session.Credentials, session.Name, err
	}

	if err = json.Unmarshal(item.Data, &session); err != nil {
		return session.Credentials, session.Name, err
	}

	return session.Credentials, session.Name, nil
}

// Store sets a profile session in the Keyring
func (s *AWSKeyringSession) SaveSessionToken(profile string, sessionName string, creds sts.Credentials) error {
	session := awsSession{
		Credentials: creds,
		Name:        AWSSessionName,
	}

	bytes, err := json.Marshal(session)
	if err != nil {
		return err
	}

	log.Infof("Saving AWS session token to keyring-item: [aws-okta: (%s) - %s]", session.Name, profile)
	s.Keyring.Set(
		keyring.Item{
			Key:                         sessionName,
			Data:                        []byte(bytes),
			Label:                       fmt.Sprintf("%s: (%s) - %s", AppName, AWSSessionName, profile),
			KeychainNotTrustApplication: false,
		},
	)

	return nil
}

func (s *AWSKeyringSession) GetMetadata(profile string) (keyring.Metadata, error) {
	md, err := s.Keyring.GetMetadata(profile)
	if err != nil {
		return keyring.Metadata{}, err
	}

	return md, nil
}

// roleSessionName calculates the role session name
func (p *AWSProvider) roleSessionName() string {
	// Try to work out a role name that will hopefully end up unique.
	return fmt.Sprintf("%d", time.Now().UTC().UnixNano())
}
