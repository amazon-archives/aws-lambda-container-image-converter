# Copyright 2019 Amazon.com, Inc. or its affiliates. All Rights Reserved.
# SPDX-License-Identifier: MIT-0

####### PHP custom runtime #######

####### Install and compile everything #######

# Same AL version as Lambda execution environment AMI
FROM amazonlinux:2017.03.1.20170812 as builder

# Lock to 2017.03 release (same as Lambda) and install compilation dependencies
RUN sed -i 's;^releasever.*;releasever=2017.03;;' /etc/yum.conf && \
    yum clean all && \
    yum install -y autoconf \
                bison \
                gcc \
                gcc-c++ \
                make \
                libcurl-devel \
                libxml2-devel \
                openssl-devel \
                tar \
                gzip \
                zip \
                unzip \
                git

# Download the PHP 7.3.0 source, compile, and install both PHP and Composer
RUN curl -sL https://github.com/php/php-src/archive/php-7.3.0.tar.gz | tar -xvz && \
    cd php-src-php-7.3.0 && \
    ./buildconf --force && \
    ./configure --prefix=/opt/php-7-bin/ --with-openssl --with-curl --with-zlib && \
    make install && \
    /opt/php-7-bin/bin/php -v && \
    curl -sS https://getcomposer.org/installer | /opt/php-7-bin/bin/php -- --install-dir=/opt/php-7-bin/bin/ --filename=composer

# Prepare runtime files
RUN mkdir -p /lambda-php-runtime/bin && \
    cp /opt/php-7-bin/bin/php /lambda-php-runtime/bin/php

COPY runtime/bootstrap /lambda-php-runtime/

# Install Guzzle, prepare vendor files
RUN mkdir /lambda-php-vendor && \
    cd /lambda-php-vendor && \
    /opt/php-7-bin/bin/php /opt/php-7-bin/bin/composer require guzzlehttp/guzzle

###### Create runtime image ######

FROM lambci/lambda:provided as runtime

# Layer 1
COPY --from=builder /lambda-php-runtime /opt/

# Layer 2
COPY --from=builder /lambda-php-vendor/vendor /opt/vendor

###### Create function image ######

FROM runtime as function

COPY function/src /var/task/src/
