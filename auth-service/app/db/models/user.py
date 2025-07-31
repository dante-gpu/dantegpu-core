import uuid
from sqlalchemy import Column, String, Boolean, DateTime, func, UUID as pgUUID
from sqlalchemy.orm import relationship
from app.db.base_class import Base
from .rbac import user_roles_table  # Import the association table


class User(Base):
    # I should define the table name.
    __tablename__ = "users"

    # I need to define the columns for the users table.
    id = Column(pgUUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    email = Column(String(255), unique=True, index=True, nullable=False)
    username = Column(String(100), unique=True, index=True, nullable=False)
    hashed_password = Column(String(255), nullable=False)
    is_active = Column(Boolean(), default=True, nullable=False)
    is_verified = Column(Boolean(), default=False, nullable=False)  # New: for email verification
    # 'role' string column is removed. Roles are now managed via user_roles_table.
    created_at = Column(DateTime(timezone=True), server_default=func.now(), nullable=False)
    updated_at = Column(DateTime(timezone=True), onupdate=func.now(), server_default=func.now(), nullable=False)
    last_login_at = Column(DateTime(timezone=True), nullable=True)  # New

    # Relationships
    profile = relationship("UserProfile", back_populates="user", uselist=False, cascade="all, delete-orphan", lazy="selectin")
    roles = relationship("Role", secondary=user_roles_table, back_populates="users", lazy="selectin")
    api_keys = relationship("UserApiKey", back_populates="user", cascade="all, delete-orphan", lazy="noload")
    # login_history can be queried separately if needed, not always loaded with user.

    # Financial fields like wallet_address, balance_dgpu, etc., are REMOVED.

    def __repr__(self):
        # I should add a representation for easier debugging.
        return f"<User(id={self.id}, email='{self.email}', username='{self.username}')>" 