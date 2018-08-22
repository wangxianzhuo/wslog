const request = require('../common/request')
const configs = require('../configs')

module.exports.getTopics = async ()=>{
    const resq = await request({
        methods: 'GET',
        uri: 'http://'+configs.configs.wslogServer+':9000/log/topics'
    })
    if (!resq.success())  {
        throw new Error('get topics error')
    }

    return resq.body
}