def minesweeper(tablero):
    """
    Convierte un tablero de Minesweeper representado por listas:
    - 0 = casilla vacía
    - 1 = mina

    Devuelve un tablero donde:
    - 9 = mina
    - 0-8 = número de minas adyacentes a esa casilla

    Entrada: lista de listas (filas) de enteros 0/1.
    Salida: nueva lista de listas con valores 0-9.
    """
    if not tablero:
        return []

    filas = len(tablero)
    cols = len(tablero[0])
    resultado = [[0] * cols for _ in range(filas)]

    for r in range(filas):
        for c in range(cols):
            if tablero[r][c] == 1:
                resultado[r][c] = 9
                continue
            cuenta = 0
            for dr in (-1, 0, 1):
                for dc in (-1, 0, 1):
                    if dr == 0 and dc == 0:
                        continue
                    nr, nc = r + dr, c + dc
                    if 0 <= nr < filas and 0 <= nc < cols and tablero[nr][nc] == 1:
                        cuenta += 1
            resultado[r][c] = cuenta

    return resultado


def imprimir_tablero(tab):
    for fila in tab:
        print(" ".join(str(x) for x in fila))


if __name__ == "__main__":
    # Ejemplo del enunciado
    ejemplo = [
        [0, 1, 0, 0],
        [0, 0, 1, 0],
        [0, 1, 0, 1],
        [1, 1, 0, 0]
    ]
    print("Entrada (ejemplo):")
    imprimir_tablero(ejemplo)
    print("\nSalida (número 9 = mina):")
    salida = minesweeper(ejemplo)
    imprimir_tablero(salida)

    # Ejemplos adicionales que se muestran por consola
    print("\nEjemplo: tablero vacío -> resultado vacío")
    empty = []
    print(minesweeper(empty))

    print("\nEjemplo: celda única con mina:")
    single_mine = [[1]]
    imprimir_tablero(single_mine)
    print("->")
    imprimir_tablero(minesweeper(single_mine))

    print("\nEjemplo: celda única vacía:")
    single_empty = [[0]]
    imprimir_tablero(single_empty)
    print("->")
    imprimir_tablero(minesweeper(single_empty))

    # Pruebas por consola (sin unittest)
    def ejecutar_prueba(nombre, entrada, esperado):
        print(f"\nPrueba: {nombre}")
        print("Entrada:")
        if entrada == []:
            print("[]")
        else:
            imprimir_tablero(entrada)
        actual = minesweeper(entrada)
        print("Salida obtenida:")
        if actual == []:
            print("[]")
        else:
            imprimir_tablero(actual)
        print("Salida esperada:")
        if esperado == []:
            print("[]")
        else:
            imprimir_tablero(esperado)
        if actual == esperado:
            print("Resultado: OK")
        else:
            print("Resultado: FALLÓ")

    ejecutar_prueba(
        "ejemplo",
        ejemplo,
        [
            [1, 9, 2, 1],
            [2, 3, 9, 2],
            [3, 9, 4, 9],
            [9, 9, 3, 1]
        ]
    )

    ejecutar_prueba("vacio", [], [])

    ejecutar_prueba("una_mina", [[1]], [[9]])

    ejecutar_prueba("una_vacia", [[0]], [[0]])

    ejecutar_prueba(
        "rectangular",
        [
            [0, 1, 0],
            [1, 0, 1]
        ],
        [
            [2, 9, 2],
            [9, 3, 9]
        ]
    )

# Explicación:
# - Versión de Python:
#   Compatible con Python 3.6+ (utiliza f-strings y comprehensions estándar).
#   Probado en Python 3.8+.
#
# - Idea principal:
#   Recorremos cada celda del tablero. Si la celda es una mina (1) la marcamos como 9
#   en el tablero de salida. Si no, contamos las minas en las 8 vecinas válidas y
#   ponemos ese contador en la celda correspondiente.
#
# - Complejidad temporal:
#   Cada celda se procesa una sola vez y para cada celda comprobamos hasta 8 vecinos,
#   por lo que la complejidad es O(n*m) donde n×m es el tamaño del tablero.
#
# - Complejidad espacial:
#   Se devuelve un tablero nuevo de tamaño n×m → O(n*m) espacio adicional.
#   Podríamos modificar el tablero en sitio para ahorrar memoria, pero eso produciría
#   efectos secundarios (la función dejaría de ser pura) y complicaría el manejo de
#   marcadores temporales (por ejemplo, distinguir minas originales de valores calculados).
#
# - Por qué esta implementación:
#   Es simple, clara y fácil de verificar. Para tableros pequeños/medianos en Python
#   es suficientemente eficiente y portable.
#
# - Alternativas:
#   * Usar numpy y convoluciones para acelerar en tableros muy grandes.
#   * Hacerlo in-place usando marcadores temporales para reducir memoria (más complejidad).
#   * Mantener una lista de coordenadas de minas y propagar incrementos a vecinas (útil
#     si hay pocas minas y el tablero es muy grande).
#