'use strict'

const request = require('request')

module.exports = function(options) {
    return new Promise((resolve, reject) => {
        request(options, (err, res, body) => {
            if(err) {
                resolve(new Response(500, {message: '请求接口服务器异常'}));
            } else {
                if(body) {
                    resolve(new Response(res.statusCode, JSON.parse(body)));
                } else {
                    resolve(new Response(res.statusCode));
                }
            }
        });
    });
};

class Response {
    constructor(status, body) {
        this.status = status;
        this.body = body;
    }

    success() {
        return this.status >= 200 && this.status < 300;
    }

    toString() {
        return '(' + this.status + ', ' + this.body + ')';
    }
}