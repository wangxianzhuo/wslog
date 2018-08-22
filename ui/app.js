const Koa = require('koa')
const Router = require('koa-router')
const logger = require('koa-logger')
const views = require('koa-views')
const configs = require('./configs')
const topic = require('./service/topic')

const app = new Koa()
const router = new Router()

app.use(logger())

app.use(views(__dirname+'/views',{
    map:{
        njk: 'nunjucks'
    }
}))

router.get('/logs', async (ctx, next)=>{
    ts = await topic.getTopics()
    await ctx.render('logs.njk', {
        topics: ts,
        wslogWrapper: configs.configs.wslogWrapper
    })
})

app.use(async (ctx, next)=>{
    var ctx = this;
    this.error = (err, status) => {
        ctx.status = status || 500;
        ctx.body = err;
    };
    await next()
})
app.use(router.routes())

app.listen('3000')
console.log('wslog ui start [:3000]')