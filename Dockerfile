####### PHP custom runtime #######

# Same AL version as Lambda execution environment AMI
FROM amazonlinux:2017.03.1.20170812 as customruntime

# Lock to 2017.03 release (same as Lambda) and install compilation dependencies
RUN sed -i 's;^releasever.*;releasever=2017.03;;' /etc/yum.conf && \
    yum clean all && \
    yum install autoconf bison gcc gcc-c++ make libcurl-devel libxml2-devel openssl-devel tar gzip -y

# Download the PHP 7.3.0 source, compile it, and install to /opt/php-7-bin
RUN mkdir ~/php-7-bin && \
    curl -sL https://github.com/php/php-src/archive/php-7.3.0.tar.gz | tar -xvz && \
    cd php-src-php-7.3.0 && \
    ./buildconf --force && \
    ./configure --prefix=/opt/php-7-bin/ --with-openssl --with-curl --with-zlib && \
    make install && \
    /opt/php-7-bin/bin/php -v

###### Something else ######

FROM lambci/lambda:provided