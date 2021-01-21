/*
AWS Okta -- https://gitlab.nordstrom.com/public-cloud/aws-okta
Maintained by Cloud Engineering <cloudengineering@nordstrom.com>
	Author: Jason Casas

Copyright 2020 @ Nordstrom, Inc. All rights reserved.
*/

package lib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
)

func ServerError(reqId string, err error) (events.APIGatewayProxyResponse, error) {

	var buf bytes.Buffer

	log.Errorf("client_request_id: %s, error: %s", reqId, err)

	values := map[string]string{"message": fmt.Sprintf("[%s] Response: (%s) - %s", reqId, http.StatusText(http.StatusInternalServerError), err)}

	jsonValue, _ := json.Marshal(values)

	json.HTMLEscape(&buf, jsonValue)

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       buf.String(),
	}, nil
}

func ClientError(reqId string, status int, err error) (events.APIGatewayProxyResponse, error) {

	var buf bytes.Buffer

	log.Errorf("client_request_id: %s, error: %s", reqId, err)

	values := map[string]string{"message": fmt.Sprintf("[%s] Response: (%s) - %s", reqId, http.StatusText(status), err)}

	jsonValue, _ := json.Marshal(values)

	json.HTMLEscape(&buf, jsonValue)

	return events.APIGatewayProxyResponse{
		StatusCode: status,
		Body:       buf.String(),
	}, nil
}
