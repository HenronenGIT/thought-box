FROM gradle:8-jdk21 AS build
WORKDIR /app
COPY . .
RUN ./gradlew --no-daemon shadowJar

FROM eclipse-temurin:21-jre-alpine
WORKDIR /app
COPY --from=build /app/build/libs/thought-box.jar /app/thought-box.jar
ENV PORT=8080
EXPOSE 8080
ENTRYPOINT ["java", "-jar", "/app/thought-box.jar"]
