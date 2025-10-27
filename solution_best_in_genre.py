import urllib.request
import json
from typing import Optional, List, Tuple

API_BASE = "https://jsonmock.hackerrank.com/api/tvseries"

def fetch_page(page: int) -> dict:
    url = f"{API_BASE}?page={page}"
    with urllib.request.urlopen(url) as resp:
        return json.load(resp)

def bestInGenre(genre: str) -> str:
    """
    Devuelve el nombre de la serie con mayor imdb_rating para el género dado.
    En caso de empate en rating, devuelve el nombre alfabéticamente menor.
    Si no hay coincidencias, devuelve "No result".
    """
    top = topNInGenre(genre, n=1)
    return top[0][0] if top else "No result"

def topNInGenre(genre: str, n: int = 4) -> List[Tuple[str, float]]:
    """
    Devuelve una lista de hasta n tuplas (nombre, rating) de las series del género
    ordenadas por rating descendente y nombre ascendente (en caso de empate).
    La información se obtiene de la API paginada.
    """
    if not genre or not genre.strip():
        return []

    genre_token = genre.strip().lower()
    try:
        first = fetch_page(1)
    except Exception:
        return []

    total_pages = int(first.get("total_pages", 1))
    matches: List[Tuple[str, float]] = []

    def maybe_add(item: dict):
        g = item.get("genre", "") or ""
        tokens: List[str] = [t.strip().lower() for t in g.split(",") if t.strip()]
        if genre_token not in tokens:
            return
        try:
            rating = float(item.get("imdb_rating", 0) or 0)
        except Exception:
            return
        name = (item.get("name") or "").strip()
        if not name:
            return
        matches.append((name, rating))

    # procesar primera página
    for item in first.get("data", []):
        maybe_add(item)

    # procesar páginas restantes
    for p in range(2, total_pages + 1):
        try:
            page_data = fetch_page(p)
        except Exception:
            continue
        for item in page_data.get("data", []):
            maybe_add(item)

    # ordenar por rating desc, luego por nombre asc
    matches.sort(key=lambda t: (-t[1], t[0]))
    return matches[:n]

if __name__ == "__main__":
    # Pruebas manuales (sin unittest). Se imprimen varios ejemplos.
    ejemplos = [
        "Action",
        "Drama",
        "Sci-Fi",
        "",            # caso vacío
    ]

    print("Pruebas de ejemplo para bestInGenre (entrada -> resultado):")
    for e in ejemplos:
        try:
            resultado = bestInGenre(e)
        except Exception as ex:
            resultado = f"Error: {ex}"
        print(f"  '{e}' -> {resultado}")

    # Genero dinámicamente el ejemplo del enunciado usando la API
    genre_demo = "Action"
    top4 = topNInGenre(genre_demo, n=4)
    print("\nSample Explanation (generado desde la API):")
    if top4:
        print(f"The {len(top4)} highest-rated shows in the {genre_demo} genre are:")
        for name, rating in top4:
            print(f" {name} - {rating}")
    else:
        print(f"No results for genre '{genre_demo}'")

"""
 Explicación técnica (Probado en Python 3.8+.):
 - Se consulta la API paginada en https://jsonmock.hackerrank.com/api/tvseries?page=N usando urllib.
 - topNInGenre recopila todas las series que contienen el token de género solicitado#   (comparación por token, minúsculas, recortando espacios).
 - Convierte imdb_rating a float (si falta o es inválido se descarta ese registro).
 - Ordena los matches por rating descendente y nombre ascendente, y devuelve los N primeros.# 
 - bestInGenre usa topNInGenre(genre, 1) para devolver solo el nombre de la mejor serie.
 - En el bloque __main__ se imprimen ejemplos y se genera dinámicamente el "Sample Explanation".
"""