package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

const (
	maxContador = 9999999
)

var (
	archivoEstado         = ""
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

func ImprimirTextoPlano(nombreImpresora string, texto string) error {
	var hPrinter uintptr

	pPrinterName, err := syscall.UTF16PtrFromString(nombreImpresora)
	if err != nil {
		return err
	}

	ret, _, err := procOpenPrinter.Call(
		uintptr(unsafe.Pointer(pPrinterName)),
		uintptr(unsafe.Pointer(&hPrinter)),
		0,
	)
	if ret == 0 {
		return fmt.Errorf("no se pudo abrir la impresora: %v", err)
	}
	defer procClosePrinter.Call(hPrinter)

	docName, _ := syscall.UTF16PtrFromString("Trabajo de Go")
	dataType, _ := syscall.UTF16PtrFromString("RAW")

	di := DOC_INFO_1{
		pDocName:    docName,
		pOutputFile: nil,
		pDatatype:   dataType,
	}

	ret, _, err = procStartDocPrinter.Call(
		hPrinter,
		1,
		uintptr(unsafe.Pointer(&di)),
	)
	if ret == 0 {
		return fmt.Errorf("error en StartDocPrinter: %v", err)
	}
	defer procEndDocPrinter.Call(hPrinter)

	ret, _, _ = procStartPagePrinter.Call(hPrinter)
	if ret == 0 {
		return fmt.Errorf("error en StartPagePrinter")
	}
	defer procEndPagePrinter.Call(hPrinter)

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

func GenerarZPL(codigo1, codigo2 string, fecha time.Time) string {

	fechaFormateada := fecha.Format("02/01/2006 15:04")

	plantillaZPL := `^XA
	^MMT
	^PW832
	^LL392
	^LS0
	^FO310,80^A0R,35,35^FD%s^FS
	^BY3,3,240^FT310,368^BCB,,N,N
	^FH\^FD>:Z>5123456>67^FS
	^PQ1,0,1,Y
	^FO010,030^A0R,50,90^FDZ%s^FS

	^FO720,80^A0R,35,35^FD%s^FS
	^BY3,3,240^FT720,368^BCB,,N,N
	^FH\^FD>:Z>5123456>67^FS
	^PQ1,0,1,Y
	^FO420,030^A0R,50,90^FDZ%s^FS
	^XZ`

	return fmt.Sprintf(plantillaZPL, fechaFormateada, codigo1, fechaFormateada, codigo2)
}

func main() {

	_, err := ObtenerImpresoraPredeterminada()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	var entrada string
	var cantidad int
	for entrada != "0" {
		entrada, err = leerTexto(`Ingrese una opción de cantidades a imprimir.

Opciones:
1- 100 etiqutas.
2- 1.000 etiqutas.
3- 3.000 etiqutas.
4- 5.000 etiqutas.
5- 10.000 etiqutas.
0- Salir
`)

		switch entrada {
		case "1":
			cantidad = 100
		case "2":
			cantidad = 1000
		case "3":
			cantidad = 3000
		case "4":
			cantidad = 5000
		case "5":
			cantidad = 10000
		case "0":
			return
		default:
			fmt.Println("Opción no valida.")
			continue
		}

		fmt.Printf("Iniciando impresión de %d códigos (%d pares)...\n", cantidad, cantidad/2)

		if err := ImprimirPares(cantidad); err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		fmt.Println("Proceso finalizado correctamente.")
	}
}

func leerTexto(mensaje string) (string, error) {
	fmt.Print(mensaje)
	entrada, err := bufio.NewReader(os.Stdin).ReadString('\n')

	if err != nil {
		return "", errors.New("Error al leer la entrada de texto")
	}

	return strings.TrimSpace(entrada), nil
}

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

	baseDir, err := os.UserConfigDir()
	if err != nil {
		fmt.Println("Error al obtener el directorio de configuración:", err)
		return err
	}

	appDir := filepath.Join(baseDir, "Nicolas-Sandoval", "default-imp-ticket")
	err = os.MkdirAll(appDir, 0755)
	if err != nil {
		fmt.Println("Error al crear el directorio de la aplicación:", err)
		return err
	}

	configPath := filepath.Join(appDir, "config")
	archivoEstado = configPath
	return os.WriteFile(configPath, fmt.Appendf(nil, "%d", num), 0644)
}

func ImprimirPares(cantidadCodigos int) error {
	impresora, err := ObtenerImpresoraPredeterminada()
	if err != nil {
		return err
	}

	ultimoNumero := LeerUltimoNumero()

	errConf := confGuardarUltimoNumero(ultimoNumero, cantidadCodigos)
	if errConf != nil {
		return err
	}
	totalPares := cantidadCodigos / 2
	contador := ultimoNumero
	for i := range totalPares {

		contador++
		if contador > maxContador {
			contador = 1
		}
		c1 := fmt.Sprintf("%07d", contador)

		contador++
		if contador > maxContador {
			contador = 1
		}
		c2 := fmt.Sprintf("%07d", contador)

		zpl := GenerarZPL(c1, c2, time.Now())
		if err := ImprimirTextoPlano(impresora, zpl); err != nil {
			return fmt.Errorf("error en par %d: %v", i+1, err)
		}

		fmt.Printf("Par %d/%d impreso: Z%s y Z%s\n", i+1, totalPares, c1, c2)
	}

	return nil
}

func confGuardarUltimoNumero(contador int, cantidadCodigos int) error {
	if err := GuardarUltimoNumero(contador + cantidadCodigos); err != nil {
		fmt.Printf("No se pudo guardar el archivo de estado: %v\n", err)
		return err
	}
	return nil
}
