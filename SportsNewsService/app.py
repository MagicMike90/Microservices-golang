import os

from flask import Flask
from models import db
from views import sports_news

app = Flask(__name__)

app_settings = os.getenv('APP_SETTINGS')
app.config.from_object(app_settings)

db.init_app(app)

app.register_blueprint(sports_news)

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=5000)
