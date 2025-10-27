import sqlite3
import pandas as pd


conn = sqlite3.connect(':memory:')
cursor = conn.cursor()


cursor.execute('''
CREATE TABLE customers (
    id SMALLINT,
    first_name VARCHAR(64),
    last_name VARCHAR(64)
)
''')

cursor.execute('''
CREATE TABLE campaigns (
    id SMALLINT,
    customer_id SMALLINT,
    name VARCHAR(64)
)
''')

cursor.execute('''
CREATE TABLE events (
    dt VARCHAR(19),
    campaign_id SMALLINT,
    status VARCHAR(64)
)
''')


customers_data = [
    (1, 'Whitney', 'Ferrero'),
    (2, 'Dickie', 'Romera')
]
cursor.executemany('INSERT INTO customers VALUES (?, ?, ?)', customers_data)


campaigns_data = [
    (1, 1, 'Upton Group'),
    (2, 1, 'Roob, Hudson and Rippin'),
    (3, 1, 'McCullough, Rempel and Larson'),
    (4, 1, 'Lang and Sons'),
    (5, 2, 'Ruecker, Hand and Haley')
]
cursor.executemany('INSERT INTO campaigns VALUES (?, ?, ?)', campaigns_data)


events_data = [
    ('2021-12-02 13:52:00', 1, 'failure'),
    ('2021-12-02 08:17:48', 2, 'failure'),
    ('2021-12-02 08:18:17', 2, 'failure'),
    ('2021-12-01 11:55:32', 3, 'failure'),
    ('2021-12-01 06:53:16', 4, 'failure'),
    ('2021-12-02 04:51:09', 4, 'failure'),
    ('2021-12-01 06:34:04', 5, 'failure'),
    ('2021-12-02 03:21:18', 5, 'failure'),
    ('2021-12-01 03:18:24', 5, 'failure'),
    ('2021-12-02 15:32:37', 1, 'success'),
    ('2021-12-01 04:23:20', 1, 'success'),
    ('2021-12-02 06:53:24', 1, 'success'),
    ('2021-12-02 08:01:02', 2, 'success'),
    ('2021-12-01 15:57:19', 2, 'success'),
    ('2021-12-02 16:14:34', 3, 'success'),
    ('2021-12-02 21:56:38', 3, 'success'),
    ('2021-12-01 05:54:43', 4, 'success'),
    ('2021-12-02 17:56:45', 4, 'success'),
    ('2021-12-02 11:56:50', 4, 'success'),
    ('2021-12-02 06:08:20', 5, 'success')
]
cursor.executemany('INSERT INTO events VALUES (?, ?, ?)', events_data)

conn.commit()


query = '''
SELECT 
    c.first_name || ' ' || c.last_name AS customer,
    COUNT(e.status) AS failures
FROM customers c
INNER JOIN campaigns cp ON c.id = cp.customer_id
INNER JOIN events e ON cp.id = e.campaign_id
WHERE e.status = 'failure'
GROUP BY c.id, c.first_name, c.last_name
HAVING COUNT(e.status) > 3
ORDER BY failures DESC
'''

result = pd.read_sql_query(query, conn)
print("\nReporte de Fallos en el Sistema de Publicidad")
print("=" * 40)
print(result.to_string(index=False))
conn.close()

"""
Requerimiento: Identificar clientes con más de 3 eventos de fallo en sus campañas

La solución requiere conectar las tres tablas (customers, campaigns, events) 
y agregar el conteo de eventos con status='failure' por cliente.

Explicación de la query:

SELECT c.first_name || ' ' || c.last_name AS customer, COUNT(e.status) AS failures
    Concateno el nombre completo del cliente usando el operador ||.
    Luego cuento todos los eventos de fallo asociados a ese cliente.

FROM customers c
INNER JOIN campaigns cp ON c.id = cp.customer_id  
INNER JOIN events e ON cp.id = e.campaign_id
    Realizo los JOINs necesarios para conectar las tablas. Utilizo INNER JOIN
    porque solo necesito registros que existan en las tres tablas.
    La relación es: customers -> campaigns -> events

WHERE e.status = 'failure'
    Filtro únicamente los eventos con status='failure', excluyendo los eventos exitosos.

GROUP BY c.id, c.first_name, c.last_name
    Agrupo por cliente para aplicar la función de agregación COUNT().
    Incluyo el id además del nombre para evitar ambigüedades en caso de nombres duplicados.

HAVING COUNT(e.status) > 3
    Filtro los grupos resultantes para incluir solo clientes con más de 3 fallos.
    Importante: HAVING actúa después del GROUP BY, mientras que WHERE actúa antes.
    La condición es estrictamente mayor (>3), no mayor o igual.

ORDER BY failures DESC
    Ordeno los resultados de forma descendente para mostrar primero 
    los clientes con mayor cantidad de fallos.

Análisis con los datos de prueba:
- Whitney Ferrero: 6 fallos totales (distribuidos en sus 4 campañas) -> Aparece en resultado
- Dickie Romera: 3 fallos exactos -> No aparece (necesita >3, no >=3)

"""