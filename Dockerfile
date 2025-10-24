FROM alpine:latest

RUN apk update && \
    apk add curl git yasm gcc make pkgconfig pcsc-lite-dev automake g++ fuse-dev vim

WORKDIR /root

RUN git clone --depth=1 -b VeraCrypt_1.26.24  https://github.com/veracrypt/VeraCrypt.git && \
    git clone -b WX_3_0_3_BRANCH https://github.com/wxWidgets/wxWidgets.git && \
    curl -L http://git.savannah.gnu.org/gitweb/?p=config.git;a=blob_plain;f=config.guess;hb=HEAD > /root/wxWidgets/config.guess && \
    curl -L http://git.savannah.gnu.org/gitweb/?p=config.git;a=blob_plain;f=config.sub;hb=HEAD > /root/wxWidgets/config.sub

RUN cd /root/VeraCrypt/src && \
    make NOGUI=1 WXSTATIC=1 WX_ROOT=/root/wxWidgets wxbuild && \
    make NOGUI=1 WXSTATIC=1 WX_ROOT=/root/wxWidgets && \
    ldd /root/VeraCrypt/src/Main/veracrypt