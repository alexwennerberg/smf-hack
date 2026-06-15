FROM php:5.6-apache
RUN set -eux; \
    printf 'deb http://archive.debian.org/debian stretch main\ndeb http://archive.debian.org/debian-security stretch/updates main\n' > /etc/apt/sources.list; \
    apt-get -o Acquire::Check-Valid-Until=false update; \
    apt-get install -y --allow-unauthenticated --no-install-recommends libpng-dev libjpeg-dev libfreetype6-dev; \
    docker-php-ext-configure gd --with-png-dir=/usr --with-jpeg-dir=/usr --with-freetype-dir=/usr; \
    docker-php-ext-install mysql gd; \
    a2enmod rewrite; \
    rm -rf /var/lib/apt/lists/*
