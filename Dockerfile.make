FROM debian:bookworm-slim
EXPOSE 3000

# Install Cairo runtime libraries. fonts-dejavu-core provides actual font files
# for Cairo text rendering (libcairo2 pulls in fontconfig/freetype but no fonts).
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    libcairo2 \
    fonts-dejavu-core \
    && rm -rf /var/lib/apt/lists/*

#ENV RMAPI_HWR_HMAC
#ENV RM_SMTP_SERVER=""
#ENV RM_SMTP_USERNAME=""
#ENV RM_SMTP_PASSWORD=""
COPY dist/rmfakecloud-docker .
ENTRYPOINT ["/rmfakecloud-docker"]
