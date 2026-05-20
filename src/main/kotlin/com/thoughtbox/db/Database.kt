package com.thoughtbox.db

import com.thoughtbox.config.DatabaseConfig
import com.zaxxer.hikari.HikariConfig
import com.zaxxer.hikari.HikariDataSource
import org.flywaydb.core.Flyway

// HikariCP is the JDBC connection pool. Node.js mental model: similar purpose to
// a pg Pool from node-postgres, but for JVM JDBC connections.
fun dataSource(config: DatabaseConfig): HikariDataSource {
    val hikari = HikariConfig()
    hikari.jdbcUrl = config.url
    hikari.username = config.user
    hikari.password = config.password
    hikari.maximumPoolSize = 5
    return HikariDataSource(hikari)
}

// Flyway scans src/main/resources/db/migration and applies new V*.sql files.
fun runMigrations(config: DatabaseConfig) {
    Flyway.configure()
        .dataSource(config.url, config.user, config.password)
        .load()
        .migrate()
}
