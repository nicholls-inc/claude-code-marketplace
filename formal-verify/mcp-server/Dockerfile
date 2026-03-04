FROM mcr.microsoft.com/dotnet/sdk:8.0 AS build
RUN dotnet tool install --global dafny --version 4.11.0
# Dafny installs as a self-contained publish, includes Z3

FROM mcr.microsoft.com/dotnet/runtime-deps:8.0-noble
COPY --from=build /root/.dotnet/tools /opt/dafny
ENV PATH="/opt/dafny:${PATH}"
RUN dafny --version
WORKDIR /work
ENTRYPOINT ["dafny"]
