# Config Package Documentation

The `config` package is responsible for managing application configuration. It provides a centralized approach to loading, validating, and accessing configuration values from embedded files, environment variables, and defaults.

---

## Overview

This package uses:
- **[Viper](https://github.com/spf13/viper)** for flexible configuration management.
- **[Go Validator](https://github.com/go-playground/validator)** to ensure all required fields are populated and valid.
- An **embedded default configuration file** (`default.ini`) as a baseline, which can be overridden by environment variables.

---

## Features

1. **Centralized Configuration**:
    - Loads default values from an embedded `default.ini` file.
    - Overrides defaults with values from environment variables.

2. **Validation**:
    - Ensures critical configuration fields are present and valid using `validator`.

3. **Flexibility**:
    - Uses the `ini` format for defaults but allows customization via environment variables.
    - Environment variables are automatically mapped by replacing `.` with `_`.

## Environment Variable Overrides

Environment variables can override `default.ini` values.
- Format: Replace `.` with `_` in variable names.
- Example:
    - `app.name` → `APP_NAME`
    - `database.password` → `DATABASE_PASSWORD`


## Best Practices

1. **Keep Secrets Secure**:  
   Avoid committing sensitive values (e.g., passwords, secrets) to version control. Use environment variables for sensitive data.

2. **Validate Configuration Early**:  
   Ensure that configuration validation is performed during application startup to catch issues early.

3. **Use Environment Variables for Deployment**:  
   Rely on environment variables to override default values in different environments (e.g., production, staging).