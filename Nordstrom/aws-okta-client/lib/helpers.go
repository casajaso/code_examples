/*
AWS Okta -- https://gitlab.nordstrom.com/public-cloud/aws-okta
Maintained by Cloud Engineering <cloudengineering@nordstrom.com>
	Author: Jason Casas

Copyright 2020 @ Nordstrom, Inc. All rights reserved.
*/

package lib

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
)

// GetEnv lookups Environmental Variable and sets default value
func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func IterStruct(s interface{}) error { //not used just testing an idea
	val := reflect.ValueOf(s).Elem()
	fieldCount := val.NumField()
	keys := make([]string, 0, fieldCount)
	for i := 0; i < fieldCount; i++ {
		//t := i.reflect.TypeOf()
		//if t == reflect.Type(struct) {
		//}
		key := val.Type().Field(i).Name
		value := val.Field(i).String()
		keys = append(keys, key+":"+value)
	}
	var jsonData []byte
	jsonData, err := json.MarshalIndent(keys, "", "    ")
	if err != nil {
		return err
	}
	fmt.Println(string(jsonData))
	return nil
}

func GetStructKeys(s interface{}) []string { //not used just testing an idea
	var keys []string
	val := reflect.ValueOf(s).Elem()
	for i := 0; i < val.NumField(); i++ {
		key := val.Type().Field(i).Name
		keys = append(keys, key)
	}
	fmt.Println(keys)
	return keys
}
