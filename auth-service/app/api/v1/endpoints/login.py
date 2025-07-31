from datetime import timedelta, datetime
from typing import Optional

from fastapi import APIRouter, Depends, HTTPException, status
from sqlalchemy.orm import Session
from jose import jwt, JWTError

from app.core.config import settings
from app.db.session import get_db
from app.db.crud import crud_user
from app.schemas.user import UserCreate
from app.schemas.token import Token, AuthResponse, LoginRequest, RegisterRequest
from app.db.models.user import User
from app.core.security import get_password_hash, verify_password

router = APIRouter()

def create_access_token(data: dict, expires_delta: Optional[timedelta] = None):
    to_encode = data.copy()
    if expires_delta:
        expire = (datetime.utcnow() + expires_delta)
    else:
        expire = datetime.utcnow() + timedelta(minutes=settings.ACCESS_TOKEN_EXPIRE_MINUTES)
    to_encode.update({"exp": expire, "type": "access"})
    encoded_jwt = jwt.encode(to_encode, settings.SECRET_KEY, algorithm=settings.ALGORITHM)
    return encoded_jwt, expire

def create_refresh_token(data: dict, expires_delta: Optional[timedelta] = None):
    to_encode = data.copy()
    if expires_delta:
        expire = datetime.utcnow() + expires_delta
    else:
        expire = datetime.utcnow() + timedelta(days=settings.REFRESH_TOKEN_EXPIRE_DAYS)
    to_encode.update({"exp": expire, "type": "refresh"})
    encoded_jwt = jwt.encode(to_encode, settings.SECRET_KEY, algorithm=settings.ALGORITHM)
    return encoded_jwt, expire


@router.post("/register", response_model=AuthResponse)
async def register(user_data: RegisterRequest, db: Session = Depends(get_db)):
    """Register a new user account"""
    existing_user_email = crud_user.get_user_by_email(db, email=user_data.email)
    if existing_user_email:
        raise HTTPException(status_code=400, detail="Email already registered")
    
    existing_user_username = crud_user.get_user_by_username(db, username=user_data.username)
    if existing_user_username:
        raise HTTPException(status_code=400, detail="Username already taken")

    user_create = UserCreate(
        username=user_data.username,
        email=user_data.email,
        password=user_data.password,
        role=user_data.role or "user"
    )
    user = crud_user.create_user(db, user_in=user_create)

    access_token, access_expires = create_access_token(
        data={"sub": user.username, "user_id": str(user.id), "email": user.email, "role": user.role}
    )
    refresh_token, _ = create_refresh_token(data={"sub": user.username, "user_id": str(user.id)})

    return AuthResponse(
        token=access_token,
        refresh_token=refresh_token,
        expires_at=access_expires,
        user_id=str(user.id),
        username=user.username,
        email=user.email,
        role=user.role,
        permissions=["user:read"]
    )

@router.post("/login", response_model=AuthResponse)
async def login(login_data: LoginRequest, db: Session = Depends(get_db)):
    """Authenticate user and return access tokens"""
    user = crud_user.authenticate_user(db, login_data.username, login_data.password)
    if not user:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid username or password",
        )
    if not user.is_active:
        raise HTTPException(status_code=400, detail="User account is disabled")

    access_token, access_expires = create_access_token(
        data={"sub": user.username, "user_id": str(user.id), "email": user.email, "role": user.role}
    )
    refresh_token, _ = create_refresh_token(data={"sub": user.username, "user_id": str(user.id)})

    return AuthResponse(
        token=access_token,
        refresh_token=refresh_token,
        expires_at=access_expires,
        user_id=str(user.id),
        username=user.username,
        email=user.email,
        role=user.role,
        permissions=["user:read"]
    ) 