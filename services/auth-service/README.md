# Auth Service (Dante Backend)

This service handles user authentication, registration, and authorization for the Dante GPU platform.

## Tech Stack

*   Python 3.10+
*   FastAPI
*   SQLAlchemy
*   PostgreSQL
*   Pydantic
*   Passlib (for password hashing)
*   python-jose (for JWTs, although token *issuance* might primarily be gateway's job)

## Setup

1.  **Create a virtual environment:**
    ```bash
    python -m venv venv
    source venv/bin/activate # or .\venv\Scripts\activate on Windows
    ```
2.  **Install dependencies:**
    ```bash
    pip install -r requirements.txt
    ```
3.  **Configure environment variables:**
    Create a `.env` file in the `auth-service` root directory (or set environment variables directly). See `app/core/config.py` for required variables (e.g., `DATABASE_URL`, `SECRET_KEY`).
    Example `.env`:
    ```
    DATABASE_URL=postgresql+psycopg2://user:password@localhost:5432/dante_auth
    SECRET_KEY=a_very_secure_secret_key_needs_to_be_long_and_random
    # Optional: ALGORITHM, ACCESS_TOKEN_EXPIRE_MINUTES
    ```
4.  **Database Setup:**
    *   Ensure you have a running PostgreSQL server.
    *   Create the database specified in `DATABASE_URL`.
    *   Run database migrations (details TBD, likely using Alembic).

## Running the Service

```bash
uvicorn app.main:app --reload --host 0.0.0.0 --port 8001 
```

This will start the FastAPI development server, typically accessible at `http://localhost:8001`.

## API Documentation

Once running, interactive API documentation (Swagger UI) is available at `http://localhost:8001/docs`.
ReDoc documentation is available at `http://localhost:8001/redoc`. 