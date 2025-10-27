// Versión de Go: 1.21+
// Esta aplicación CLI resume archivos de texto usando la API de Inferencia gratuita de HuggingFace
// Documentación de la API: https://huggingface.co/docs/api-inference/quicktour
// Modelo usado: facebook/bart-large-cnn
// Página del modelo: https://huggingface.co/facebook/bart-large-cnn
// Tipo de tarea: Summarization (text-summarization)
//
// AUTENTICACIÓN:
// Aunque la API es gratuita, requiere un token de API para su uso.
// Se puede obtener un token gratuito en: https://huggingface.co/settings/tokens
// 
// Explicacion de como configurar la variable de entorno 
// 
// PowerShell (opción con comillas escapadas):
//   $env:HUGGINGFACE_API_TOKEN = 'tu_token_aqui'
// 
// CMD:
//   set HUGGINGFACE_API_TOKEN=tu_token_aqui
// 
// Linux/Mac:
//   export HUGGINGFACE_API_TOKEN=tu_token_aqui
//
// MODELOS ALTERNATIVOS (si bart-large-cnn no funciona):
// - sshleifer/distilbart-cnn-12-6 (más rápido, menos preciso)
// - google/pegasus-xsum (excelente para resúmenes muy cortos)
// - t5-base (modelo multipropósito de Google)

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// HuggingFaceRequest representa el payload de la solicitud para la API de HuggingFace
type HuggingFaceRequest struct {
	Inputs     string                 `json:"inputs"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// HuggingFaceResponse representa la respuesta de la API
type HuggingFaceResponse []struct {
	SummaryText string `json:"summary_text"`
}

// HuggingFaceError representa las respuestas de error de la API
type HuggingFaceError struct {
	Error string `json:"error"`
}

const (
	// Endpoint de la API de Inferencia de HuggingFace para el modelo de resumen BART
	// El modelo facebook/bart-large-cnn está optimizado para resumir noticias y artículos
	// Página del modelo: https://huggingface.co/facebook/bart-large-cnn
	apiURL = "https://api-inference.huggingface.co/models/facebook/bart-large-cnn"

	// Longitud máxima de entrada para evitar límites de la API
	maxInputLength = 1024

	// Configuración de reintentos para solicitudes a la API
	maxRetries        = 3
	initialRetryDelay = 2 * time.Second
)

func main() {
	// Verificar token de API
	apiToken := os.Getenv("HUGGINGFACE_API_TOKEN")
	if apiToken == "" {
		fmt.Println("Error: No se encontró el token de HuggingFace API")
		fmt.Println("")
		fmt.Println("Para usar esta herramienta, necesitas un token gratuito de HuggingFace:")
		fmt.Println("1. Ve a: https://huggingface.co/settings/tokens")
		fmt.Println("2. Crea un nuevo token (cuenta gratuita)")
		fmt.Println("3. Configura la variable de entorno:")
		fmt.Println("")
		fmt.Println("   PowerShell (sin comillas internas):")
		fmt.Println("   $env:HUGGINGFACE_API_TOKEN = \"tu_token_aqui\"")
		fmt.Println("")
		fmt.Println("   PowerShell (con comillas simples):")
		fmt.Println("   $env:HUGGINGFACE_API_TOKEN = 'tu_token_aqui'")
		fmt.Println("")
		fmt.Println("   CMD:")
		fmt.Println("   set HUGGINGFACE_API_TOKEN=tu_token_aqui")
		fmt.Println("")
		fmt.Println("   Linux/Mac:")
		fmt.Println("   export HUGGINGFACE_API_TOKEN=tu_token_aqui")
		fmt.Println("")
		fmt.Println("4. Verifica con: echo $env:HUGGINGFACE_API_TOKEN")
		os.Exit(1)
	}

	// Define CLI flags
	var summaryType string
	var inputFile string

	flag.StringVar(&summaryType, "type", "medium", "Summary type: short, medium, or bullet")
	flag.StringVar(&summaryType, "t", "medium", "Summary type: short, medium, or bullet (shorthand)")
	flag.StringVar(&inputFile, "input", "", "Path to the text file to summarize")

	flag.Parse()

	// Handle positional argument if --input not provided
	if inputFile == "" {
		args := flag.Args()
		if len(args) > 0 {
			inputFile = args[0]
		} else {
			fmt.Println("Error: No input file specified")
			fmt.Println("Usage: go run solution_summarizer.go --input <file> --type <short|medium|bullet>")
			fmt.Println("   or: go run solution_summarizer.go -t <short|medium|bullet> <file>")
			os.Exit(1)
		}
	}

	// Validate summary type
	summaryType = strings.ToLower(summaryType)
	if summaryType != "short" && summaryType != "medium" && summaryType != "bullet" {
		fmt.Printf("Error: Invalid summary type '%s'. Must be: short, medium, or bullet\n", summaryType)
		os.Exit(1)
	}

	// Leer el archivo de entrada
	content, err := readFile(inputFile)
	if err != nil {
		fmt.Printf("Error reading file '%s': %v\n", inputFile, err)
		os.Exit(1)
	}

	// Truncar contenido si es muy largo
	if len(content) > maxInputLength {
		content = content[:maxInputLength]
		fmt.Fprintf(os.Stderr, "Warning: Input truncated to %d characters\n", maxInputLength)
	}

	// Generar resumen
	summary, err := summarizeText(content, summaryType, apiToken)
	if err != nil {
		fmt.Printf("Error generating summary: %v\n", err)
		os.Exit(1)
	}

	// Mostrar el resumen
	fmt.Println(summary)
}

// readFile lee todo el contenido de un archivo de texto
func readFile(filePath string) (string, error) {
	// Verificar si el archivo existe
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", fmt.Errorf("file does not exist: %s", filePath)
	}

	// Leer contenido del archivo
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Asegurar que el archivo no esté vacío
	content := strings.TrimSpace(string(data))
	if content == "" {
		return "", fmt.Errorf("file is empty")
	}

	return content, nil
}

// summarizeText llama a la API de HuggingFace para generar un resumen según el tipo especificado
// Implementa lógica de reintentos con backoff exponencial para manejar límites de tasa y errores transitorios
func summarizeText(text, summaryType, apiToken string) (string, error) {
	var lastErr error

	// Bucle de reintentos con backoff exponencial
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Calcular retraso de backoff exponencial
			delay := initialRetryDelay * time.Duration(1<<uint(attempt-1))
			fmt.Fprintf(os.Stderr, "Retrying in %v... (attempt %d/%d)\n", delay, attempt+1, maxRetries)
			time.Sleep(delay)
		}

		summary, err := attemptSummarization(text, summaryType, apiToken)
		if err == nil {
			return summary, nil
		}

		lastErr = err

		// Verificar si el error es reintentable (límite de tasa o error de servidor)
		if !isRetryableError(err) {
			return "", err
		}
	}

	return "", fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}

// attemptSummarization realiza un único intento de llamar a la API
func attemptSummarization(text, summaryType, apiToken string) (string, error) {
	// Preparar el prompt según el tipo de resumen
	prompt := buildPrompt(text, summaryType)

	// Crear payload de solicitud
	requestBody := HuggingFaceRequest{
		Inputs: prompt,
		Parameters: map[string]interface{}{
			"max_length": getMaxLength(summaryType),
			"min_length": getMinLength(summaryType),
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Crear solicitud HTTP
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiToken)

	// Crear cliente HTTP con timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Ejecutar solicitud
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	// Leer cuerpo de la respuesta
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Verificar errores de la API
	if resp.StatusCode != http.StatusOK {
		var errResp HuggingFaceError
		if json.Unmarshal(body, &errResp) == nil && errResp.Error != "" {
			apiErr := &APIError{
				StatusCode: resp.StatusCode,
				Message:    errResp.Error,
			}
			// Mejorar mensaje de error 401 con instrucciones útiles
			if resp.StatusCode == http.StatusUnauthorized {
				return "", fmt.Errorf("%w\n\nPlease ensure your API token is valid:\n1. Go to https://huggingface.co/settings/tokens\n2. Create or copy your token\n3. Set: $env:HUGGINGFACE_API_TOKEN=\"your_token_here\"", apiErr)
			}
			return "", apiErr
		}
		return "", &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}

	// Parsear respuesta
	var response HuggingFaceResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(response) == 0 || response[0].SummaryText == "" {
		return "", fmt.Errorf("no summary generated by the API")
	}

	// Formatear la salida según el tipo de resumen
	summary := response[0].SummaryText
	return formatOutput(summary, summaryType), nil
}

// APIError representa un error devuelto por la API con código de estado
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error (%d): %s", e.StatusCode, e.Message)
}

// isRetryableError determina si vale la pena reintentar un error
func isRetryableError(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		// Reintentar en límite de tasa (429) o errores de servidor (5xx)
		return apiErr.StatusCode == http.StatusTooManyRequests ||
			(apiErr.StatusCode >= 500 && apiErr.StatusCode < 600)
	}
	return false
}

// buildPrompt crea un prompt adaptado al tipo de resumen
func buildPrompt(text, summaryType string) string {
	switch summaryType {
	case "short":
		return fmt.Sprintf("Summarize this text in 1-2 concise sentences:\n\n%s", text)
	case "medium":
		return fmt.Sprintf("Provide a comprehensive paragraph summary of this text:\n\n%s", text)
	case "bullet":
		return fmt.Sprintf("Summarize this text as a list of key points:\n\n%s", text)
	default:
		return text
	}
}

// getMaxLength devuelve la longitud máxima de tokens para el tipo de resumen
func getMaxLength(summaryType string) int {
	switch summaryType {
	case "short":
		return 50
	case "medium":
		return 150
	case "bullet":
		return 200
	default:
		return 100
	}
}

// getMinLength devuelve la longitud mínima de tokens para el tipo de resumen
func getMinLength(summaryType string) int {
	switch summaryType {
	case "short":
		return 10
	case "medium":
		return 50
	case "bullet":
		return 30
	default:
		return 20
	}
}

// formatOutput formatea el resumen según el tipo solicitado
func formatOutput(summary, summaryType string) string {
	if summaryType == "bullet" {
		// Convertir a puntos bullet si no está ya formateado
		// Maneja múltiples delimitadores: puntos, saltos de línea y punto y coma
		var bullets []string
		
		// Intentar dividir por saltos de línea primero (si la API devuelve lista pre-formateada)
		lines := strings.Split(summary, "\n")
		if len(lines) > 1 {
			for _, line := range lines {
				line = strings.TrimSpace(line)
				// Remover marcadores bullet existentes si están presentes
				line = strings.TrimPrefix(line, "-")
				line = strings.TrimPrefix(line, "*")
				line = strings.TrimPrefix(line, "•")
				line = strings.TrimSpace(line)
				if line != "" && len(line) > 3 { // Evitar fragmentos muy cortos
					bullets = append(bullets, "- "+line)
				}
			}
		}
		
		// Si no se encontraron saltos de línea, dividir por puntos o punto y coma
		if len(bullets) == 0 {
			// Dividir tanto por puntos como por punto y coma
			text := strings.ReplaceAll(summary, ";", ".")
			lines = strings.Split(text, ".")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line != "" && len(line) > 10 { // Evitar fragmentos muy cortos
					bullets = append(bullets, "- "+line)
				}
			}
		}
		
		if len(bullets) > 0 {
			return strings.Join(bullets, "\n")
		}
		// Fallback: devolver original con bullet único si el parseo falla
		return "- " + summary
	}
	return summary
}

/*
================================================================================
DESCRIPCIÓN DEL CÓDIGO Y DECISIONES DE DISEÑO
================================================================================

RESUMEN GENERAL:
Esta aplicación CLI proporciona capacidades de resumen de texto utilizando la
API de Inferencia de HuggingFace. Lee un archivo de texto, lo envía a un modelo
GenAI (facebook/bart-large-cnn) y genera un resumen formateado según el tipo
de resumen solicitado.

DECISIONES CLAVE DE DISEÑO:

1. SELECCIÓN DE API:
   - Se eligió la API de Inferencia de HuggingFace por su tier gratuito
   - Se seleccionó el modelo facebook/bart-large-cnn: modelo de última generación
     específicamente entrenado para artículos de noticias y texto general
   - Requiere token de API gratuito (obtenible en huggingface.co/settings/tokens)
   - El token se pasa vía variable de entorno para seguridad

2. PARSEO DE ARGUMENTOS CLI:
   - Se utilizó el paquete estándar "flag" de Go para parseo CLI nativo e idiomático
   - Soporta tanto flags nombrados (--input, --type) como abreviados (-t)
   - Permite argumentos posicionales como alternativa para mayor flexibilidad UX
   - Proporciona mensajes de uso claros y valida todas las entradas

3. INGENIERÍA DE PROMPTS:
   - Se implementó una función dedicada buildPrompt() para personalizar prompts por tipo
   - Cada tipo de resumen recibe instrucciones específicas para guiar a la IA:
     * short: Solicita explícitamente "1-2 oraciones concisas"
     * medium: Solicita "resumen de párrafo completo"
     * bullet: Solicita "lista de puntos clave"
   - Se combinó la ingeniería de prompts con parámetros de API (max_length, min_length)
     para asegurar formatos de salida consistentes

4. ESTRATEGIA DE MANEJO DE ERRORES:
   - Validación comprehensiva en cada paso: existencia de archivo, archivos vacíos,
     errores de API, parseo de respuestas
   - Se implementó el tipo personalizado APIError para información estructurada de errores
   - Mensajes de error amigables que guían a los usuarios a resolver problemas
   - Lógica de reintentos con backoff exponencial para fallos transitorios
   - Distingue entre errores reintentables (429, 5xx) y no reintentables

5. LÓGICA DE REINTENTOS CON BACKOFF EXPONENCIAL:
   - Implementa hasta 3 intentos de reintento para llamadas a la API
   - Usa backoff exponencial (2s, 4s, 8s) para evitar saturar la API
   - Solo reintenta en rate limit (429) o errores de servidor (500-599)
   - Falla rápido en errores de cliente (400-499 excepto 429) para ahorrar tiempo
   - Proporciona feedback al usuario durante los intentos de reintento

6. DISEÑO DEL CLIENTE HTTP:
   - Usa el paquete estándar net/http de Go por confiabilidad
   - Timeout de 30 segundos previene colgarse en solicitudes lentas/fallidas
   - Limpieza apropiada de recursos con defer resp.Body.Close()
   - Establece el header Content-Type correcto para solicitudes JSON
   - Separa la lógica HTTP en attemptSummarization() para manejo limpio de reintentos

7. FORMATEO DE SALIDA:
   - Función formatOutput() mejorada maneja múltiples casos edge
   - Para puntos bullet: intenta múltiples estrategias de parseo:
     * Primero intenta items separados por saltos de línea (la API puede pre-formatear)
     * Se repliega a separación por puntos/punto y coma
     * Remueve marcadores de bullet existentes para evitar duplicación
     * Filtra fragmentos muy cortos (< 10 chars) para mantener calidad
   - Proporciona fallback sensato si todo el parseo falla
   - Formato de salida limpio y consistente para todos los tipos de resumen

8. MANEJO DE ENTRADA:
   - Trunca la entrada a 1024 caracteres para respetar límites de la API
   - Advierte al usuario en stderr cuando ocurre truncado (no contamina stdout)
   - Valida existencia de archivo antes de intentar leer
   - Asegura que el archivo no esté vacío para evitar llamadas de API desperdiciadas

9. ORGANIZACIÓN DEL CÓDIGO:
   - Clara separación de responsabilidades con funciones enfocadas
   - Structs para request/response de API proporcionan seguridad de tipos
   - Constantes para configuración facilitan el ajuste
   - Funciones helper (getMaxLength, getMinLength) encapsulan lógica

10. EXTENSIBILIDAD:
    - Fácil agregar nuevos tipos de resumen (solo actualizar switch statements)
    - El endpoint de API puede cambiarse modificando una sola constante
    - Parámetros de reintento configurables vía constantes
    - Lógica de formateo de salida aislada para fácil modificación

COMPROMISOS (TRADE-OFFS):

- Truncado de entradas largas: Se eligió simplicidad sobre chunking/combinar resúmenes
  (chunking requeriría lógica más compleja y múltiples llamadas a la API)
  
- Backoff exponencial: Comienza en 2s lo cual puede sentirse lento, pero previene
  throttling de la API y sigue mejores prácticas para APIs públicas
  
- Formateo bullet: Múltiples estrategias de parseo agregan complejidad pero manejan
  varios formatos de respuesta de la API 

- Sin streaming: Espera respuesta completa en lugar de streaming de tokens
  (implementación más simple, adecuada para caso de uso de resumen)

================================================================================
*/
