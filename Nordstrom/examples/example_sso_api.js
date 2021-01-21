#!/usr/bin/env node

// Author: Jason Casas - Nordstrom Public Cloud
// nodejs example using custom credential_process to retrieve credentials
// Requires aws-sdk v1.16.0 or above to support custom credential_process

const AWS = require('aws-sdk');
const os = require('os');
const fs = require('fs');

getSSOCache = async () => {
    return new Promise(function(resolve, reject) {
        let home_dir = os.homedir()
        let rawdata = fs.readFileSync(`${home_dir}/.aws/sso/cache/0c2eea704ee08603d192a6e22220cb618db79d17.json`);
        let data = JSON.parse(rawdata);
        resolve(data)
    })
}

authenticateUser = async () => { 
    let ssoCache = await getSSOCache()
    let sso = new AWS.SSO({region: ssoCache.region})
    let params = {
        accessToken: ssoCache.accessToken, 
        accountId: '116019673048', 
        roleName: 'AdministratorAccess'
    }
    return new Promise(function(resolve, reject) {
        sso.getRoleCredentials(params, function(error, response) {
            if (error) {
                console.log(error, error.stack)
                reject(new Error(error.stack))
            } else {
                AWS.config.credentials = response.roleCredentials
                resolve()
            }            
        })
    })
}

getUserIdentity = async () => {
    return new Promise(function(resolve, reject) {
        let sts = new AWS.STS();
        sts.getCallerIdentity(function(error, response) {
            if (error) {
                reject(new Error(error.stack))
            }
            delete response['ResponseMetadata']
            resolve(response)
        })
    })
}
 
(async () => {
    await authenticateUser()
    let identity = await getUserIdentity()
    console.log(identity)
})().catch(error => {
    console.error(error)
});