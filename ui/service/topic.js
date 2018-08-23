const request = require('../common/request')
const configs = require('../config')

module.exports.getTopics = async () => {
    const resq = await request({
        methods: 'GET',
        uri: 'http://' + configs.configs.wslogServer + '/log/topics'
    })
    if (!resq.success()) {
        throw new Error('获取topics异常')
    }

    return resq.body
}