const WebSocket = require('ws')

var connect = function (addr) {
  let session = new WebSocket(addr)
  console.log('connect to [' + addr + ']')

  session.on('ping', function ping() {
    session.pong('pong')
    console.log(new Date() + ' send pong')
  })

  session.on('pong', function pong() {
    console.log(new Date() + ' receive pong')
  })
  return session
}
var ping = function (pingPeriod, session) {
  let id = setInterval(function ping() {
    if (session && session.readyState === session.OPEN) {
      session.ping()
      console.log(new Date() + ' send ping')
    } else {
      clearInterval(id)
    }
  }, pingPeriod)
}

module.exports.connect = connect
module.exports.ping = ping
