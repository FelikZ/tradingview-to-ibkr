package main

import (
    "bufio"
    "fmt"
    "os"
    "strings"
    "flag"
)

var lseetfTickers = map[string]bool{
    "IB01": true,
    "IBTA": true,
    "VDTA": true,
    "URNU": true,
}

func main() {
    inputFilePath, outputFilePath := parseFlags()
    err := convertWatchlist(inputFilePath, outputFilePath)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
    fmt.Printf("Conversion completed. Output saved to %s\n", outputFilePath)
}

func parseFlags() (string, string) {
    flag.Usage = func() {
        fmt.Fprintf(os.Stderr, "Usage: %s <input_file> <output_file>\n", os.Args[0])
        fmt.Fprintf(os.Stderr, "Convert TradingView watchlist to IBKR format\n")
        fmt.Fprintf(os.Stderr, "\nArguments:\n")
        fmt.Fprintf(os.Stderr, "  input_file   Path to the input TradingView watchlist file\n")
        fmt.Fprintf(os.Stderr, "  output_file  Path to the output IBKR watchlist file\n")
        fmt.Fprintf(os.Stderr, "\nFlags:\n")
        flag.PrintDefaults()
    }

    flag.Parse()

    if flag.NArg() != 2 {
        flag.Usage()
        os.Exit(1)
    }

    return flag.Arg(0), flag.Arg(1)
}

func convertWatchlist(inputFilePath, outputFilePath string) error {
    inputFile, err := os.Open(inputFilePath)
    if err != nil {
        return fmt.Errorf("error opening input file: %v", err)
    }
    defer inputFile.Close()

    outputFile, err := os.Create(outputFilePath)
    if err != nil {
        return fmt.Errorf("error creating output file: %v", err)
    }
    defer outputFile.Close()

    outputFile.WriteString("CSVEXPORT\nCOLUMN,0\n")

    scanner := bufio.NewScanner(inputFile)
    for scanner.Scan() {
        err := processLine(scanner.Text(), outputFile)
        if err != nil {
            return err
        }
    }

    if err := scanner.Err(); err != nil {
        return fmt.Errorf("error reading input file: %v", err)
    }

    return nil
}

func processLine(line string, outputFile *os.File) error {
    symbols := strings.Split(line, ",")
    for _, symbol := range symbols {
        if strings.HasPrefix(symbol, "###") {
            section := strings.TrimPrefix(symbol, "###")
            _, err := outputFile.WriteString(fmt.Sprintf("HED,%s\n", section))
            if err != nil {
                return fmt.Errorf("error writing section header: %v", err)
            }
            continue
        }

        err := processSymbol(symbol, outputFile)
        if err != nil {
            return err
        }
    }
    return nil
}

func processSymbol(symbol string, outputFile *os.File) error {
    parts := strings.Split(symbol, ":")
    if len(parts) != 2 {
        return nil // Skip invalid symbols
    }

    exchange := parts[0]
    ticker := parts[1]

    var err error
    switch exchange {
    case "FX", "FX_IDC":
        err = handleCurrencyPair(outputFile, ticker)
    case "TVC":
        err = handleTVCSymbol(outputFile, ticker)
    case "FRED":
        _, err = outputFile.WriteString(fmt.Sprintf("DES,%s,IND,SMART,,,,,\n", ticker))
    case "CBOE":
        _, err = outputFile.WriteString(fmt.Sprintf("DES,%s,IND,CBOE,,,,,\n", ticker))
    case "ECONOMICS":
        // Skip economics indicators
    default:
        err = handleDefaultSymbol(outputFile, exchange, ticker)
    }

    if err != nil {
        return fmt.Errorf("error processing symbol %s: %v", symbol, err)
    }
    return nil
}

func handleCurrencyPair(outputFile *os.File, ticker string) error {
    currencies := strings.Split(ticker, "USD")
    if len(currencies) != 2 {
        return nil // Skip invalid currency pairs
    }

    base := currencies[0]
    quote := currencies[1]
    
    if base == "GBP" || base == "EUR" {
        _, err := outputFile.WriteString(fmt.Sprintf("DES,%s,CASH,IDEALPRO,,,,,USD\n", base))
        return err
    }
    
    if quote == "" {
        quote = base
    }
    if quote == "CNY" {
        quote = "HKD"
    }
    _, err := outputFile.WriteString(fmt.Sprintf("DES,USD,CASH,IDEALPRO,,,,,%s\n", quote))
    return err
}

func handleTVCSymbol(outputFile *os.File, ticker string) error {
    var output string
    switch ticker {
    case "DXY":
        output = "DES,DX,IND,NYBOT,,,,,\n"
    case "US10Y":
        output = "DES,TNX,IND,CBOE,,,,,\n"
    case "US05Y":
        output = "DES,FVX,IND,CBOE,,,,,\n"
    default:
        if strings.HasPrefix(ticker, "US") && strings.HasSuffix(ticker, "Y") {
            output = fmt.Sprintf("DES,%s,IND,SMART,,,,,\n", ticker)
        } else {
            return nil // Skip other TVC symbols
        }
    }
    _, err := outputFile.WriteString(output)
    return err
}

func handleDefaultSymbol(outputFile *os.File, exchange, ticker string) error {
    if ticker == "BRK.B" {
        ticker = "BRK B"
    }
    var output string
    if isLSEETFTicker(ticker) {
        output = fmt.Sprintf("DES,%s,STK,SMART/LSEETF,,,,,\n", ticker)
    } else if isAmericanETForStock(exchange) {
        output = fmt.Sprintf("DES,%s,STK,SMART/AMEX,,,,,\n", ticker)
    } else {
        output = fmt.Sprintf("DES,%s,STK,SMART,,,,,\n", ticker)
    }
    _, err := outputFile.WriteString(output)
    return err
}

func isAmericanETForStock(exchange string) bool {
    americanExchanges := []string{"NYSE", "NASDAQ", "AMEX"}
    for _, e := range americanExchanges {
        if e == exchange {
            return true
        }
    }
    return false
}

func isLSEETFTicker(ticker string) bool {
    return lseetfTickers[ticker]
}