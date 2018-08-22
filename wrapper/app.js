const Koa = require('koa')
const logger = require('koa-logger')
const Router = require('koa-router')
const websockify = require('koa-websocket')
const configs = require('./config')

const app = websockify(new Koa())
const router = new Router()

var ws = require('./ws')

router.get('/log/:topic', (ctx, next) => {
  let session = {}
  let topic = ctx.params.topic
  if (topic === '') {
    topic = 'log_msg'
  }
  let querys = ctx.querystring
  if (querys) {
    querys = ('?' + querys)
  }
  try {
    session = ws.connect('ws://'+configs.wslogServer + '/ws/log' + topic + querys)
  } catch (err) {
    console.error(err)
    next()
    return
  }
  ws.ping(10000, session)
  session.on('message', function (message) {
    ctx.websocket.send(message)
    console.log(new Date() + ' send message: [' + message + ']')
  })

  session.on('close', function (code, reason) {
    if (session != null) {
      session.close()
      session = null
      console.log(new Date() + ' close code [' + code + '], reason [' + reason + ']')
    }
    // close client ws
    ctx.websocket.close()
  })

  session.on('open', function () {
    session.ping()
    console.log(new Date() + ' send ping')
  })

  session.on('error', function error (error) {
    console.error(new Date() + ' error: ' + error)
    if (session != null) {
      session.close()
    }
    // close client ws
    ctx.websocket.close()
  })

  // receive client close event
  ctx.websocket.on('close', function (code, reason) {
    if (session != null) {
      session.close()
    }
    console.log(new Date() + ' client close code [' + code + '], reason [' + reason + ']')
  })

  next()
})

app.ws.use(logger())
app.ws.use(router.routes())

app.listen('3000')
console.log('wslog wrapper start [:3000]')
