FROM joseluisq/static-web-server:2-alpine
COPY dist/ /public
EXPOSE 80
