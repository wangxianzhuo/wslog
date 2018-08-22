FROM node:8-alpine

RUN mkdir -p /usr/app
WORKDIR /usr/app

COPY package.json /usr/app/
RUN npm install -g cnpm --registry=https://registry.npm.taobao.org
RUN cnpm install 

COPY . /usr/app/

EXPOSE 3000
CMD ["npm", "start"]