# docker build -t mgrast/awe-monitor .
# docker run -ti --rm --name awe-monitor -p 8085:80 -v `pwd`/config.js:/usr/share/nginx/html/js/config.js mgrast/awe-monitor


FROM nginx:1.13-alpine


COPY . /usr/share/nginx/html

