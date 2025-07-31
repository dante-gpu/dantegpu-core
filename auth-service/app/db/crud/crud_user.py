from sqlalchemy.orm import Session
from typing import Optional
import uuid

from app.db.models.user import User
from app.schemas.user import UserCreate, UserUpdate
from app.core.security import get_password_hash, verify_password # I need the hashing and verification functions


# === Get Operations ===

def get_user(db: Session, user_id: uuid.UUID) -> Optional[User]:
    """I need a function to get a single user by their ID."""
    return db.query(User).filter(User.id == user_id).first()


def get_user_by_email(db: Session, email: str) -> Optional[User]:
    """I need a function to get a single user by their email address."""
    return db.query(User).filter(User.email == email).first()


def get_user_by_username(db: Session, username: str) -> Optional[User]:
    """I need a function to get a single user by their username."""
    return db.query(User).filter(User.username == username).first()


def get_users(
    db: Session, skip: int = 0, limit: int = 100
) -> list[User]:
    """I need a function to get a list of users, with pagination (skip, limit)."""
    return db.query(User).offset(skip).limit(limit).all()


# === Create Operation ===

def create_user(db: Session, *, user_in: UserCreate) -> User:
    """I need a function to create a new user in the database."""
    # I must hash the password before storing it.
    hashed_password = get_password_hash(user_in.password)

    # I should create the SQLAlchemy model instance.
    # Note: Pydantic's model_dump (or dict) excludes unset fields by default.
    # We need all fields from UserCreate except the plain password.
    db_user = User(
        email=user_in.email,
        username=user_in.username,
        hashed_password=hashed_password,
        is_active=True,  # Default to active on creation
        role=user_in.role
    )

    # I need to add the new user object to the session and commit.
    db.add(db_user)
    db.commit()

    # I should refresh the object to get database-generated values (like ID, created_at).
    db.refresh(db_user)
    return db_user


# === Update Operation ===

def update_user(
    db: Session, *, db_user: User, user_in: UserUpdate
) -> User:
    """I need a function to update an existing user."""
    # I should get the update data from the input schema.
    # exclude_unset=True ensures only provided fields are used for update.
    update_data = user_in.model_dump(exclude_unset=True)

    # If a new password was provided, I need to hash it.
    if update_data.get("password"):
        hashed_password = get_password_hash(update_data["password"])
        update_data["hashed_password"] = hashed_password
        del update_data["password"] # Don't store the plain password field

    # I need to iterate over the update data and set the attributes on the DB object.
    for field, value in update_data.items():
        setattr(db_user, field, value)

    # I should add the updated object to the session and commit.
    db.add(db_user)
    db.commit()

    # Refresh to get any updated fields from the DB (like updated_at).
    db.refresh(db_user)
    return db_user


# === Delete Operation ===

def delete_user(db: Session, *, user_id: uuid.UUID) -> Optional[User]:
    """I need a function to delete a user by their ID."""
    db_user = db.query(User).filter(User.id == user_id).first()
    if db_user:
        db.delete(db_user)
        db.commit()
    return db_user


# === Authentication Helper (Placeholder) ===
# While full authentication logic belongs in an API layer or service,
# a CRUD helper might exist.
def authenticate_user(db: Session, username: str, password: str) -> Optional[User]:
    """
    I need a function to check if a user exists and the password is correct.
    Note: This mixes concerns slightly, authentication logic is often separate.
    Returns the user object if authentication succeeds, otherwise None.
    """
    user = get_user_by_username(db, username=username)
    if not user:
        return None
    
    if not user.is_active:
        # Maybe return a specific error or flag for inactive users?
        return None

    if not verify_password(password, user.hashed_password):
        return None

    return user