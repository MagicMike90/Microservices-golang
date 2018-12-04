import os

from flask import Flask

app = Flask(__name__)

app_settings = os.getenv('APP_SETTINGS')
app.config.from_object(app_settings)


if __name__ == '__main__':
    app.run(host='0.0.0.0', port=5000)
