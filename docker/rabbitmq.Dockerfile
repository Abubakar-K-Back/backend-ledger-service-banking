# docker/rabbitmq.Dockerfile
FROM rabbitmq:3.12-management

RUN mkdir -p /var/lib/rabbitmq
RUN chown -R rabbitmq:rabbitmq /var/lib/rabbitmq