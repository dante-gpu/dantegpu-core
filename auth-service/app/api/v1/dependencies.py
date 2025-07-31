from fastapi import Depends, HTTPException, status, Security
from fastapi.security import HTTPBearer, HTTPAuthorizationCredentials
from sqlalchemy.orm import Session
from jose import JWTError, jwt

from app.core.config import Settings
from app.db.session import get_db
from app.db.models.user import User
from app.db.crud.crud_user import get_user_by_username
from app.schemas.token import TokenData

security = HTTPBearer()
settings = Settings()

def verify_token(token: str) -> TokenData:
    try:
        payload = jwt.decode(token, settings.SECRET_KEY, algorithms=[settings.ALGORITHM])
        
        username = payload.get("sub")
        user_id = payload.get("user_id")
        email = payload.get("email")
        role = payload.get("role")

        if not all([username, user_id, email, role]):
            raise HTTPException(
                status_code=status.HTTP_401_UNAUTHORIZED,
                detail="Invalid token: payload missing required fields",
                headers={"WWW-Authenticate": "Bearer"},
            )
            
        token_data = TokenData(
            username=str(username),
            user_id=str(user_id),
            email=str(email),
            role=str(role)
        )
        return token_data
    except JWTError:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid authentication credentials",
            headers={"WWW-Authenticate": "Bearer"},
        )

def get_current_user(
    credentials: HTTPAuthorizationCredentials = Security(security),
    db: Session = Depends(get_db)
) -> User:
    token = credentials.credentials
    token_data = verify_token(token)
    
    if not token_data.username:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid token: username missing",
            headers={"WWW-Authenticate": "Bearer"},
        )
    
    current_username: str = token_data.username
    user = get_user_by_username(db, username=current_username)
    if user is None:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="User not found",
            headers={"WWW-Authenticate": "Bearer"},
        )
    return user

def get_current_active_user(current_user: User = Depends(get_current_user)) -> User:
    if not current_user.is_active:
        raise HTTPException(status_code=400, detail="Inactive user")
    return current_user 