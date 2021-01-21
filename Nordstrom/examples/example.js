#!/usr/bin/env node
 
const AWS = require('aws-sdk');
 
getCredentials = async () => {
    return new Promise(function(resolve, reject) {
        process.env.AWS_SDK_LOAD_CONFIG = true;
        var creds = new AWS.ProcessCredentials({profile: 'nordstrom-federated'});
        AWS.config.credentials = creds;
        AWS.config.region = 'us-west-2'
        AWS.config.getCredentials(function(error, response) {
            if (error) {
                reject(new Error(error.stack))
            }
            resolve(response)
        })
    })
}
  
listBuckets = async () => {
    return new Promise(function(resolve, reject) {
        var s3 = new AWS.S3();
        var bucketInfo = s3.listBuckets(function(error, response) {
            if (error) {
                reject(new Error(error.stack))
            }
            resolve(response)
        })
    })
}

getIdentity = async () => {
    return new Promise(function(resolve, reject) {
        var sts = new AWS.STS();
        var identity = sts.getCallerIdentity(function(error, response) {
            if (error) {
                reject(new Error(error.stack))
            }
            delete response['ResponseMetadata']
            resolve(response)
        })
    })
}
  
(async () => {
    await getCredentials()
    var buckets = await listBuckets()
    console.log(buckets)
})().catch(error => {
    console.error(error)
});