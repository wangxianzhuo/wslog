const Koa = require('koa')
const Router = require('koa-router')
const logger = require('koa-logger')
const views = require('koa-views')
const configs = require('./config')
const topic = require('./service/topic')

const app = new Koa()
const router = new Router()

app.use(logger())

app.use(views(__dirname + '/views', {
    map: {
        njk: 'nunjucks'
    }
}))

router.get('/logs', async (ctx, next) => {
    try {
        ts = await topic.getTopics()
        await ctx.render('logs.njk', {
            topics: ts,
            wslogWrapper: configs.configs.wslogWrapper
        })
    } catch (err) {
        console.log(err)
        ctx.status = 500
        ctx.body = err.message
    }
})

app.use(router.routes())

app.listen(configs.configs.address)
console.log('wslog ui start [' + configs.configs.address + ']')