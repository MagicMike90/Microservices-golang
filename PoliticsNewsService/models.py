import os
from datetime import datetime

from mongoengine import (DateTimeField, Document, IntField, ListField,
                         StringField, connect)
from sqlalchemy import BigInteger, Column, DateTime, Index, String
from sqlalchemy.dialects import postgresql
from sqlalchemy.ext.declarative import declarative_base

Base = declarative_base()


class CommandNewsModel(Base):
    __tablename__ = 'news'

    id = Column(BigInteger, primary_key=True)
    version = Column(BigInteger, primary_key=True)
    title = Column(String(length=200))
    content = Column(String)
    author = Column(String(length=50))
    created_at = Column(DateTime, default=datetime.utcnow)
    published_at = Column(DateTime)
    news_type = Column(String, default='politics')
    tags = Column(postgresql.ARRAY(String))

    __table_args__ = Index('index', 'id', 'version'),


connect('politics', host=os.environ.get('QUERYBD_HOST'))


class QueryNewsModel(Document):
    id = IntField(primary_key=True)
    version = IntField(required=True)
    title = StringField(required=True, max_length=200)
    content = StringField(required=True)
    author = StringField(required=True, max_length=50)
    created_at = DateTimeField(default=datetime.utcnow)
    published_at = DateTimeField()
    news_type = StringField(default="politics")
    tags = ListField(StringField(max_length=50))
