FROM joseluisq/static-web-server:2-alpine
COPY dist/ /var/public
EXPOSE 80
