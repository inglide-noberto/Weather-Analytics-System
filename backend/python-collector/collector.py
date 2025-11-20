import os
import time
import requests
import json
import pika
import schedule
from datetime import datetime

# Conexão RabbitMQ
RABBITMQ_URI = os.getenv('RABBITMQ_URI', 'amqp://guest:guest@localhost:5672/')
QUEUE_NAME = 'weather_logs_queue'

# Configuração da API de Clima
API_KEY = os.getenv('WEATHER_API_KEY')
CITY_LAT = os.getenv('CITY_LAT')
CITY_LON = os.getenv('CITY_LON')

INTERVAL_HOURS = int(os.getenv('COLLECTION_INTERVAL_HOURS', 1))

WEATHER_URL = f"https://api.openweathermap.org/data/3.0/onecall?lat={CITY_LAT}&lon={CITY_LON}&exclude=minutely,hourly,daily&appid={API_KEY}&units=metric&lang=pt_br"

def fetch_weather_data():
    """Busca dados climáticos e retorna um objeto normalizado."""
    print(f"[{datetime.now().isoformat()}] Coletando dados de clima para {CITY_LAT}, {CITY_LON}...")
    
    if not API_KEY or not CITY_LAT or not CITY_LON:
        print("ERRO: Variáveis de ambiente da API de clima estão faltando.")
        return None

    try:
        response = requests.get(WEATHER_URL)
        response.raise_for_status()
        data = response.json()
        
        current = data.get('current', {})

        normalized_data = {
            'location_lat': CITY_LAT,
            'location_lon': CITY_LON,
            'timestamp': datetime.now().isoformat(),
            'temperature': current.get('temp'),
            'humidity': current.get('humidity'),
            'wind_speed': current.get('wind_speed'),
            'condition': current.get('weather', [{}])[0].get('description', 'Desconhecido'),
            'cloudiness': current.get('clouds', 0),
        }
        
        return normalized_data

    except requests.exceptions.RequestException as e:
        print(f"ERRO ao buscar dados da API: {e}")
        return None
    except Exception as e:
        print(f"ERRO inesperado na coleta: {e}")
        return None

# --- FUNÇÃO DE PUBLICAÇÃO NO RABBITMQ ---

def publish_to_queue(message):
    """Conecta ao RabbitMQ e publica a mensagem JSON."""
    try:
        params = pika.URLParameters(RABBITMQ_URI)
        connection = pika.BlockingConnection(params)
        channel = connection.channel()
        
        # Garante que a fila exista (se não existir, cria)
        channel.queue_declare(queue=QUEUE_NAME, durable=True) 
        
        # Publica a mensagem com persistência
        channel.basic_publish(
            exchange='',
            routing_key=QUEUE_NAME,
            body=message.encode('utf-8'),
            properties=pika.BasicProperties(
                delivery_mode=2, # Torna a mensagem persistente
            )
        )
        print(f"--> [SUCESSO] Mensagem enviada para RabbitMQ: {QUEUE_NAME}")
        connection.close()

    except pika.exceptions.AMQPConnectionError as e:
        print(f"ERRO: Não foi possível conectar ao RabbitMQ ({RABBITMQ_URI}). O serviço está rodando? {e}")
    except Exception as e:
        print(f"ERRO na publicação: {e}")

# --- FUNÇÃO PRINCIPAL E SCHEDULER ---

def job():
    """Tarefa que coleta e publica os dados."""
    data = fetch_weather_data()
    if data:
        json_message = json.dumps(data)
        publish_to_queue(json_message)
    else:
        print("Coleta de dados falhou, pulando a publicação.")

def main():
    print("--- Python Collector Iniciado ---")
    print(f"Agendando coleta a cada {INTERVAL_HOURS} hora(s)...")

    job() 
    
    schedule.every(INTERVAL_HOURS).hours.do(job)

    while True:
        schedule.run_pending()
        time.sleep(1)

if __name__ == '__main__':
    main()