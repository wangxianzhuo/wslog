module.exports.configs = {
    wslogWrapper:  process.env.WS_LOG_WRAPPER || 'ws.log.wrapper:3000/log',
    wslogServer: process.env.WS_LOG_SERVER || 'ws.log.server:9000'
}