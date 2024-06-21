FROM docker:cli

WORKDIR /usr/src/app
ENV PORT=80
COPY dist/mandok /usr/src/app/mandok
CMD ["./mandok"]