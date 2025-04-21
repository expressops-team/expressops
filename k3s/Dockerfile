FROM nginx:alpine

RUN apk update

ENV TZ=Europe/Madrid

RUN apk add --no-cache tzdata && \
    cp /usr/share/zoneinfo/$TZ /etc/localtime && \
    echo $TZ > /etc/timezone && \
    apk del tzdata

RUN echo 'Ejemplo de POD con Kubernetes y Yaml' > /usr/share/nginx/html/index.html

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]


