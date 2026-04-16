# Docker image to build stcopenmpp from the latest version of Ubuntu
# Usage:
#   docker build \ 
#     --build-arg OMPP_GIT_URL=https://github.com/statcan/stcopenmpp \
#     -t stcopenmpp-build:ubuntu .

# Declare base image
FROM ubuntu:latest

# This tells tools like apt-get not to prompt for user input
ARG DEBIAN_FRONTEND=noninteractive

# Update base image
RUN apt-get update
RUN apt-get install -y apt-utils
RUN apt-get upgrade -y

# # Install build tools and runtime dependencies for native compilation and tooling
RUN apt-get install -y g++ && \
  apt-get install -y make && \
  apt-get install -y bison flex && \
  apt-get install -y git && \
  apt-get install -y sqlite3 && \
  apt-get install -y openmpi-bin libopenmpi-dev && \
  apt-get install -y curl && \
  apt-get install -y xz-utils

# Download and install Go
RUN curl -L "https://go.dev/dl/go1.25.5.linux-amd64.tar.gz" \
  --output /tmp/go_setup.tar.gz && \
  tar -xzf /tmp/go_setup.tar.gz -C /tmp && \
  mv /tmp/go / && \
  rm -rf /tmp/gocache /tmp/tmp && \
  rm /tmp/go_setup.tar.gz

# Download and install unixODBC (optional)
RUN apt-get install -y unixodbc unixodbc-dev

# Clean up
RUN apt-get autoclean

# Download and install Node.js
RUN curl -L "https://nodejs.org/dist/v24.14.0/node-v24.14.0-linux-x64.tar.xz" \
  --output /tmp/node.tar.xz && \
  mkdir /node && \
  tar -xJf /tmp/node.tar.xz -C /node --strip-components=1 && \
  rm /tmp/node.tar.xz
  
# Set local timezone
RUN rm -f /etc/localtime && \
  ln -s /usr/share/zoneinfo/America/Toronto /etc/localtime

# Copy entrypoint and build scripts
COPY entrypoint.sh \
  build-all \
  build-openm \
  build-models \
  build-go \
  build-ui \
  build-tar-xz \
  model.ini \
  make-doc \
  make-model \
  README.md \
  /scripts/

RUN chmod 755 /scripts/entrypoint.sh && \
  chmod 755 /scripts/build-all && \
  chmod 755 /scripts/build-openm && \
  chmod 755 /scripts/build-models && \
  chmod 755 /scripts/build-go && \
  chmod 755 /scripts/build-ui && \
  chmod 755 /scripts/build-tar-xz && \
  chmod 755 /scripts/make-doc && \
  chmod 744 /scripts/model.ini && \
  chmod 755 /scripts/make-model && \
  chmod 744 /scripts/README.md

# Set image labels
LABEL name=stcopenmpp-build:ubuntu
LABEL os=Linux
LABEL license=MIT
LABEL description="stcopenmpp build environemnt: GNU Compiler Collection (GCC), GNU Make, Open MPI, Git, SQLite, GNU Bison, Flex, Go, MinGW, node.js"

# Set image build variables
ARG OMPP_GIT_URL=https://github.com/statcan/stcopenmpp
ARG OMPP_USER=ompp

# Expose ARGs as environment variables
ENV OMPP_GIT_URL=${OMPP_GIT_URL}
ENV OMPP_USER=${OMPP_USER}

# Set default runtime user configuration
ENV OMPP_USER=build \
  OMPP_GROUP=build \
  OMPP_UID=1000 \
  OMPP_GID=1000 \
  OMPP_BUILD_TAG=dev

# Entry point initializes all required build environments:
ENTRYPOINT ["/scripts/entrypoint.sh"]

# Default command: print the README to guide users who run the container without arguments
CMD ["cat", "/scripts/README.md"]
