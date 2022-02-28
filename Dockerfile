FROM shekyan/slowhttptest@sha256:36655254ac7a3b5859b74cfe370767d4f553d5fc2926ff46ce870c8ae698779b as slowhttptest
FROM utkudarilmaz/hping3:latest
ARG DD_API_URL
ENV DD_API_URL=$DD_API_URL
COPY bin/main /opt/main
COPY --from=slowhttptest /usr/local/bin/slowhttptest /usr/local/bin/slowhttptest
COPY ./scripts/infinite.sh /opt/infinite.sh
WORKDIR /opt
ENTRYPOINT ./main