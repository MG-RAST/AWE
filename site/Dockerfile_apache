
# docker build -t mgrast/awe-monitor .

# apache: docker run -ti --rm --name awe-monitor -p 8085:80 -v `pwd`/config.js:/usr/local/apache2/htdocs/js/config.js mgrast/awe-monitor
# nginx:  docker run -ti --rm --name awe-monitor -p 8085:80 -v `pwd`/config.js:/usr/share/nginx/html/js/config.js mgrast/awe-monitor


# needed: /usr/local/apache2/cgi-bin/AuthConfig.pm

FROM httpd:2.4-alpine


COPY . /usr/local/apache2/htdocs/
RUN mv /usr/local/apache2/htdocs/httpd.conf /usr/local/apache2/conf
RUN apk update ; apk add \
	perl-cgi \
	perl-json \
	perl-libwww

ADD https://raw.githubusercontent.com/MG-RAST/authServer/master/client/authclient.cgi /usr/local/apache2/cgi-bin/authclient.cgi
RUN chmod 755 /usr/local/apache2/cgi-bin/authclient.cgi


# nginx, but does not support cgi (required for auth client)
#FROM nginx:1.13-alpine
#COPY . /usr/share/nginx/html



