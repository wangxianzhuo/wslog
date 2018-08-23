module.exports.configs = {
    wslogWrapper: process.env.WS_LOG_WRAPPER || 'wrapper.ws.log:3000/log',
    wslogServer: process.env.WS_LOG_SERVER || 'server.ws.log:9000',
    address: process.env.ADDRESS || '3000'
}