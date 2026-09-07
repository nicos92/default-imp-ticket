package main

import (
	"fmt"
	"syscall"
	"unsafe"
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

func main() {
	fmt.Println("Buscando impresora predeterminada...")

	nombreImpresora, err := ObtenerImpresoraPredeterminada()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Impresora predeterminada: '%s'\n", nombreImpresora)

	// Texto de prueba para imprimir (incluimos salto de línea y avance de página/corte)
	mensaje := "^XA^MMT^PW751^LL392^LS0^BY3,3,240^FT240,368^BCB,,N,N^FH\\^FD>:Z>5123456>67^FS^PQ1,0,1,Y^XZ            \n\n\n\x0C" // \x0C es Form Feed (Avanzar hoja)

	fmt.Println("Enviando texto a la impresora...")
	err = ImprimirTextoPlano(nombreImpresora, mensaje)
	if err != nil {
		fmt.Printf("Error al imprimir: %v\n", err)
		return
	}

	fmt.Println("¡Documento enviado con éxito a la cola de impresión!")
}
