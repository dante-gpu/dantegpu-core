from sqlalchemy.orm import DeclarativeBase
from sqlalchemy import MetaData
from typing import Any

# I should define a naming convention for constraints for Alembic migrations.
# This helps keep migration files consistent and avoids unnamed constraints.
convention = {
    "ix": "ix_%(column_0_label)s",
    "uq": "uq_%(table_name)s_%(column_0_name)s",
    "ck": "ck_%(table_name)s_%(constraint_name)s",
    "fk": "fk_%(table_name)s_%(column_0_name)s_%(referred_table_name)s",
    "pk": "pk_%(table_name)s",
}

metadata = MetaData(naming_convention=convention)

# I need to create the base class for my SQLAlchemy models.
class Base(DeclarativeBase):
    metadata = metadata
    
    # Optionally, define type annotation map or other base configurations
    # type_annotation_map = {dict[str, Any]: JSON}
    pass 