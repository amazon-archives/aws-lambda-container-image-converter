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

# Download the PHP 7.3.0 source, compile, and install
RUN mkdir ~/php-7-bin && \
    curl -sL https://github.com/php/php-src/archive/php-7.3.0.tar.gz | tar -xvz && \
    cd php-src-php-7.3.0 && \
    ./buildconf --force && \
    ./configure --prefix=/opt/php-7-bin/ --with-openssl --with-curl --with-zlib && \
    make install && \
    /opt/php-7-bin/bin/php -v

# Prepare runtime files
WORKDIR /runtime
RUN mkdir bin && \
    cp /opt/php-7-bin/bin/php bin/php

RUN curl -sS https://getcomposer.org/installer | ./bin/php -- --install-dir=/runtime/bin --filename=composer && \
    ./bin/php ./bin/composer require guzzlehttp/guzzle

###### Create runtime image ######

FROM lambci/lambda:provided as runtime

COPY --from=builder /runtime/ /opt/

COPY runtime/bootstrap /opt/

###### Create function image ######

FROM runtime as function

COPY function/ /var/task/src/
