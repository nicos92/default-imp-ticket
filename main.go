package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

const (
	archivoEstado = "ultimo_codigo.txt"
	maxContador   = 9999999
)

var (
	winspool              = syscall.NewLazyDLL("winspool.drv")
	procGetDefaultPrinter = winspool.NewProc("GetDefaultPrinterW")
	procOpenPrinter       = winspool.NewProc("OpenPrinterW")
	procClosePrinter      = winspool.NewProc("ClosePrinter")
	procStartDocPrinter   = winspool.NewProc("StartDocPrinterW")
	procEndDocPrinter     = winspool.NewProc("EndDocPrinter")
	procStartPagePrinter  = winspool.NewProc("StartPagePrinter")
	procEndPagePrinter    = winspool.NewProc("EndPagePrinter")
	procWritePrinter      = winspool.NewProc("WritePrinter")
)

// Estructura requerida por Windows para describir el trabajo de impresión
type DOC_INFO_1 struct {
	pDocName    *uint16
	pOutputFile *uint16
	pDatatype   *uint16
}

func ObtenerImpresoraPredeterminada() (string, error) {
	var size uint32
	procGetDefaultPrinter.Call(0, uintptr(unsafe.Pointer(&size)))
	if size == 0 {
		return "", fmt.Errorf("no se pudo determinar el tamaño")
	}

	buffer := make([]uint16, size)
	ret, _, err := procGetDefaultPrinter.Call(
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(unsafe.Pointer(&size)),
	)
	if ret == 0 {
		return "", fmt.Errorf("error al obtener impresora: %v", err)
	}

	return syscall.UTF16ToString(buffer), nil
}

// ImprimirTextoPlano envía una cadena de texto a la impresora especificada
func ImprimirTextoPlano(nombreImpresora string, texto string) error {
	var hPrinter uintptr

	// 1. Convertir nombre de impresora a UTF-16
	pPrinterName, err := syscall.UTF16PtrFromString(nombreImpresora)
	if err != nil {
		return err
	}

	// 2. Abrir la impresora
	ret, _, err := procOpenPrinter.Call(
		uintptr(unsafe.Pointer(pPrinterName)),
		uintptr(unsafe.Pointer(&hPrinter)),
		0,
	)
	if ret == 0 {
		return fmt.Errorf("no se pudo abrir la impresora: %v", err)
	}
	defer procClosePrinter.Call(hPrinter)

	// 3. Configurar la información del documento (Formato RAW)
	docName, _ := syscall.UTF16PtrFromString("Trabajo de Go")
	dataType, _ := syscall.UTF16PtrFromString("RAW")

	di := DOC_INFO_1{
		pDocName:    docName,
		pOutputFile: nil,
		pDatatype:   dataType,
	}

	// 4. Iniciar el documento
	ret, _, err = procStartDocPrinter.Call(
		hPrinter,
		1,
		uintptr(unsafe.Pointer(&di)),
	)
	if ret == 0 {
		return fmt.Errorf("error en StartDocPrinter: %v", err)
	}
	defer procEndDocPrinter.Call(hPrinter)

	// 5. Iniciar la página
	ret, _, _ = procStartPagePrinter.Call(hPrinter)
	if ret == 0 {
		return fmt.Errorf("error en StartPagePrinter")
	}
	defer procEndPagePrinter.Call(hPrinter)

	// 6. Escribir los datos en la impresora
	bytesTexto := []byte(texto)
	var bytesEscritos uint32

	ret, _, err = procWritePrinter.Call(
		hPrinter,
		uintptr(unsafe.Pointer(&bytesTexto[0])),
		uintptr(len(bytesTexto)),
		uintptr(unsafe.Pointer(&bytesEscritos)),
	)
	if ret == 0 {
		return fmt.Errorf("error al escribir datos: %v", err)
	}

	return nil
}

// GenerarZPLCrea la plantilla ZPL con los datos dinámicos
func GenerarZPL(codigo1, codigo2 string, fecha time.Time) string {
	// Formateamos la fecha en español/estándar (ej: 07/09/2026 14:30)
	fechaFormateada := fecha.Format("02/01/2006 15:04")

	// Usamos fmt.Sprintf para reemplazar las variables %s en el ZPL
	plantillaZPL := `^XA
	^MMT
	^PW751
	^LL392
	^LS0
	^FO290,80^A0R,35,35^FD%s^FS
	^BY3,3,240^FT290,368^BCB,,N,N
	^FH\^FD>:Z>5123456>67^FS
	^PQ1,0,1,Y
	^FO0,30^A0R,50,90^FDZ%s^FS

	^FO680,80^A0R,35,35^FD%s^FS
	^BY3,3,240^FT680,368^BCB,,N,N
	^FH\^FD>:Z>5123456>67^FS
	^PQ1,0,1,Y
	^FO380,30^A0R,50,90^FDZ%s^FS
	^XZ`

	return fmt.Sprintf(plantillaZPL, fechaFormateada, codigo1, fechaFormateada, codigo2)
}

func main() {
	fmt.Println("Buscando impresora predeterminada...")

	nombreImpresora, err := ObtenerImpresoraPredeterminada()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Impresora predeterminada: '%s'\n", nombreImpresora)

	// Texto de prueba para imprimir (incluimos salto de línea y avance de página/corte)
	// mensajeUno := "^XA^MMT^PW751^LL392^LS0^BY3,3,240^FT240,368^BCB,,N,N^FH\\^FD>:Z>5123456>67^FS^PQ1,0,1,Y^XZ            \n\n\n\x0C" // \x0C es Form Feed (Avanzar hoja)

	// fechaActual := time.Now()
	// mensaje := GenerarZPL("1234567", fechaActual)
	// fmt.Println("Enviando texto a la impresora...")
	// err = ImprimirTextoPlano(nombreImpresora, mensaje)
	// if err != nil {
	// 	fmt.Printf("Error al imprimir: %v\n", err)
	// 	return
	// }

	// fmt.Println("¡Documento enviado con éxito a la cola de impresión!")

	// Ejemplo: Solicitar impresión de 10 códigos (5 pares)
	cantidadAImprimir := 4

	fmt.Printf("Iniciando impresión de %d códigos (%d pares)...\n", cantidadAImprimir, cantidadAImprimir/2)

	if err := ImprimirPares(cantidadAImprimir); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("Proceso finalizado correctamente.")
}

// 3. Manejo del Estado (Lectura/Escritura)
func LeerUltimoNumero() int {
	data, err := os.ReadFile(archivoEstado)
	if err != nil {
		return 0
	}
	num, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}
	return num
}

func GuardarUltimoNumero(num int) error {
	return os.WriteFile(archivoEstado, []byte(fmt.Sprintf("%d", num)), 0644)
}

// 5. Función Principal de Impresión en Pares
func ImprimirPares(cantidadCodigos int) error {
	impresora, err := ObtenerImpresoraPredeterminada()
	if err != nil {
		return err
	}

	contador := LeerUltimoNumero()
	totalPares := cantidadCodigos / 2

	for i := range totalPares {
		// Primer código del par
		contador++
		if contador > maxContador {
			contador = 1
		}
		c1 := fmt.Sprintf("%07d", contador)

		// Segundo código del par
		contador++
		if contador > maxContador {
			contador = 1
		}
		c2 := fmt.Sprintf("%07d", contador)

		// Generar y enviar ZPL
		zpl := GenerarZPL(c1, c2, time.Now())
		if err := ImprimirTextoPlano(impresora, zpl); err != nil {
			return fmt.Errorf("error en par %d: %v", i+1, err)
		}

		// Guardar progreso
		if err := GuardarUltimoNumero(contador); err != nil {
			fmt.Printf("No se pudo guardar el archivo de estado: %v\n", err)
		}

		fmt.Printf("Par %d/%d impreso: Z%s y Z%s\n", i+1, totalPares, c1, c2)
	}

	return nil
}
