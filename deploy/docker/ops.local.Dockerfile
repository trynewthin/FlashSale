FROM flashsale-backend:local

WORKDIR /app
ENV TZ=Asia/Shanghai

USER root

RUN apk add --no-cache docker-cli docker-cli-compose git

COPY .memory/ops-build/fs /app/bin/fs

RUN mkdir -p /app/log

EXPOSE 9100

ENTRYPOINT ["/app/bin/fs", "ops", "server"]
CMD ["--addr", "0.0.0.0:9100", "--repo-root", "/app"]
