// 09/2018 // 
// Dumps CSV formatted cognito user data pool for cross-region backup and replication //

const AWS = require('aws-sdk');
const fs  = require('fs');
const path  = require('path');
const JSONStream = require('JSONStream');
const { promisify } = require('util');
const Limiter = require('./Limiter')

const stringify = JSONStream.stringify();
const awsRegion = 'us-west-2'
const cognito = new AWS.CognitoIdentityServiceProvider({region: awsRegion});
const listPools = promisify(cognito.listUserPools).bind(cognito);
const listUsers = promisify(cognito.listUsers).bind(cognito);
const directory = '/tmp/'

!fs.existsSync(directory) && fs.mkdirSync(directory)
const limiter = new Limiter(null, 200);

//finds userpools in current AWS account. returns: [ { all poolAttributes } ]
const listUserPools = async () => {
    try {
        const { UserPools } = await listPools({MaxResults: 60});
        return(UserPools);
    }
    catch (err) {
        console.error(`[ERROR][listUserPools] ${(err)}`);
    }
};

//reduce pool info returns array: [ {poolName, poolId} ] 
const gatherPoolIds = async () => {
    const userPools = await listUserPools();
    const reducePoolInfo = userPools.reduce((acc, current) => {
        keyVals = {
            Name:current.Name, 
            Id:current.Id
        };
        acc.push(keyVals);
        return acc;
    }, []);
    return(reducePoolInfo);
};

//gather user info by poolId pool returns: { userPoolId } { userAttributes }
const listUserAttrib = async (params) => {
    const isoDate = new Date(Date()).toISOString();
    const fileName = (params.UserPoolId + isoDate);
    const file = path.join(directory, fileName);
    const writeStream = fs.createWriteStream(file);
    let count = 0;
    try {
        stringify.pipe(writeStream);
        const paginateUsers = async (params) => {
            const { Users , PaginationToken } = await limiter.limiter.schedule(() => listUsers(params));
            console.log(limiter.limiter.counts());
            Users.forEach((user) => {
                stringify.write(user);
            });
            if (PaginationToken) {
                params.PaginationToken = PaginationToken;
                return paginateUsers(params);
            } else {
                stringify.end();
                stringify.on('end', () => {
                    writeStream.end();
                });
                console.log(`USER #${++count}`);
                console.log('DONE')
                return true;
            }
        };
        await paginateUsers(params);
    }
    catch (err) {
        console.error(`[ERROR][listUsers] ${(err)}`);
    }
};

//itterates poolId retruns: params { userPoolId } 
const listUsersAllPools = async () => { 
    const poolIds = await gatherPoolIds();
    poolIds.forEach((pool) => {
        let userPoolId = (pool.Id);
        let params = {
            UserPoolId: userPoolId
        };
        listUserAttrib(params, userPoolId);
    });
};

//hard-coding pool-id temporarily to reduce run time/cost by avoiding iter-pools//
const poolId = 'us-west-2_npk0wWZAH';

if ( poolId == '' ) {
    listUsersAllPools();
} 
else {
    const params = {
    UserPoolId: poolId
    };

    listUserAttrib(params);
}