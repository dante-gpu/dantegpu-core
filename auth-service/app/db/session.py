from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker
from app.core.config import settings

# I need the DATABASE_URL from settings
# I need to create the SQLAlchemy engine.
# connect_args is useful for SQLite, but for PostgreSQL it's usually empty.
engine = create_engine(
    settings.DATABASE_URL,
    pool_pre_ping=True  # Checks connection validity before use
    # connect_args={"check_same_thread": False}  # Only needed for SQLite
)

# I need to create a configured "Session" class.
SessionLocal = sessionmaker(autocommit=False, autoflush=False, bind=engine)

# --- Dependency for FastAPI ---
# I should create a dependency function to get a DB session per request.
# This ensures the session is always closed after the request.
def get_db():
    db = SessionLocal()
    try:
        yield db
    finally:
        db.close() 