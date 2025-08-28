package tinyq

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type SecretToken struct {
	Name       string
	Permission string
	GenerateOn time.Time
	Decoded    string
	IsValid    bool
	IsReadonly bool
	Issue      string
	ExpiresOn  time.Time
}

func removePadding(input string) string {
	return strings.TrimLeft(input, "0")
}

func addpadding(str, pad string, total int) string {
	for len(str) < total {
		str = pad + str
	}
	return str
}

func ValidateToken(decoded, salt string) *SecretToken {
	decodedToken := DecodeToken(decoded, salt)
	fmt.Println("decoded token", decodedToken)
	if decodedToken == "" {
		fmt.Println("empty token")
		return &SecretToken{Issue: "empty token"}
	}

	if len(decodedToken) != 42 {
		fmt.Println("len < | > 42", len(decodedToken))
		return &SecretToken{Issue: "invalid token (size)"}
	}

	header := decodedToken[:5]
	if header != "TINYQ" {
		fmt.Println("invalid header")
		return &SecretToken{Issue: "invalid token (header)"}
	}

	//name is next 19 character
	name := decodedToken[5:24]
	finalname := removePadding(name)
	// for _, char := range name {
	// 	if char == '0' {
	// 		continue
	// 	}
	// 	finalname += string(char)
	// }

	//permission is next 2 character
	permission := decodedToken[24:26]
	if permission != "RR" && permission != "RW" {
		return &SecretToken{Issue: "invalid token (permission)"}
	}

	//date is next 8 character
	datepart := decodedToken[26:38]
	generatedon, err := time.Parse("010220061504", datepart)
	if err != nil {
		return &SecretToken{Issue: "invalid token (date)"}
	}
	fmt.Println("date Part", datepart)

	hourspart := decodedToken[38:42]
	hours, err := strconv.Atoi(removePadding(hourspart))
	if err != nil {
		return &SecretToken{Issue: "invalid token (hours)"}
	}
	fmt.Println("Hours Part", hourspart)
	expireson := generatedon.Add(time.Duration(hours) * time.Hour)

	return &SecretToken{
		Name:       finalname,
		Permission: permission,
		GenerateOn: generatedon,
		Decoded:    decodedToken,
		IsValid:    time.Since(generatedon) < time.Hour*24,
		IsReadonly: permission == "RR",
		ExpiresOn:  expireson,
	}
}

func GenerateTokenString(name, permission string, hours int) string {
	header := "TINYQ"
	namewithpadding := addpadding(name, "0", 19)
	// namelength := len(name)
	// padding := 19 - namelength
	now := time.Now().Format("010220061504")
	hourswithpadding := addpadding(strconv.Itoa(hours), "0", 4)
	fmt.Println("Hours with padding", hourswithpadding)
	str := header + namewithpadding + permission + now + hourswithpadding
	return str
}

func generateSalt() string {
	return uuid.New().String()[0:6]
}

// ZERkY2VlZ2FkRWVjY0BkRGRkZGFnZ2REZGRkY2RhZmhkRGZBZ2NnZ2RpZWNjaGdnZEFlY2ZGZkFkRmdoZGFlQGRDZmljZGREZGZkYWNkZWJkZGZnZ2dkRmRlZWFnZ2RFZGRnZ2dnZERkZmViZWVlZg==
func GenerateToken(name, permission string, hours int) (string, string) {
	start := time.Now()
	salt := generateSalt()
	token := GenerateTokenString(name, permission, hours)
	// total := calculatetotal(salt)
	// total = total / 12
	reversedToken := reverse(token)
	fmt.Println("reversed", reversedToken)
	// var str string
	// for _, r := range reversedToken {
	// 	// fmt.Println(r, string(r))
	// 	f := r - 9
	// 	str += string(f)
	// }

	// str := reverse(reversedToken)

	str := base64.StdEncoding.EncodeToString([]byte(reversedToken)) //5
	fmt.Println("based", str)

	// compressed := CompressSmallString(token)
	hexed := hex.EncodeToString([]byte(str)) //4
	fmt.Println("hexed", hexed)

	shifted := shiftNumber(hexed)
	fmt.Println("shifted", shifted)
	finaltoken := convertuppertolowerandlowetoupper(shifted) // 2 3
	fmt.Println("uppertolower", finaltoken)

	fmt.Println("took", time.Since(start))
	// fmt.Println("final based", base64.StdEncoding.EncodeToString([]byte(finaltoken)))
	based2 := base64.StdEncoding.EncodeToString([]byte(finaltoken)) //1
	fmt.Println("based2 (final)", based2)
	return based2, salt
}

func DecodeToken(encodedtoken, salt string) string {
	fmt.Println("original", encodedtoken)
	unbasedtoken, _ := base64.StdEncoding.DecodeString(encodedtoken) //1
	fmt.Println("unbased2", string(unbasedtoken))
	unbasedtokenstr := convertuppertolowerandlowetoupper(string(unbasedtoken)) //2
	fmt.Println("lowetoupper", unbasedtokenstr)
	// unshifted := shiftASCII(unbasedtokenstr) //3
	// fmt.Println("unshifted", unshifted)

	unhexed, err := hex.DecodeString(unbasedtokenstr) //4
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("unhexed", string(unhexed))

	// uncompressed := DecompressSmallString(string(unhexed))
	unbase64ed, _ := base64.StdEncoding.DecodeString(string(unhexed)) //5
	fmt.Println("unbase64ed", string(unbase64ed))

	// total := calculatetotal(salt)
	// total = total / 12
	reversedToken := reverse(string(unbase64ed)) //6
	fmt.Println("reversed", reversedToken)
	// fmt.Println("dec", str)
	return reversedToken
}

func convertuppertolowerandlowetoupper(str string) string {
	var finalstring string
	for _, r := range str {
		if r >= 'A' && r <= 'Z' {
			finalstring += string(r + 32)
		} else if r >= 'a' && r <= 'z' {
			finalstring += string(r - 32)
		} else {
			finalstring += string(r)
		}
	}
	return finalstring
}

func shiftNumber(str string) string {

	// fmt.Println("shift", str)
	var result string
	for _, r := range str {
		if r >= '0' && r <= '9' {
			n, _ := strconv.Atoi(string(r))

			// fmt.Println(65 + n)
			result += string(65 + n)
		} else {
			result += string(r)
		}
	}
	return result
}

func shiftASCII(str string) string {
	// fmt.Println("unshift1", str)
	var result string
	for _, r := range str {
		if r >= 'A' && r <= 'Z' {
			n := 65 - int(r)
			result += strconv.Itoa(n)
		} else {
			result += string(r)
		}
	}
	// fmt.Println("unshift2", result)
	return result
}

func calculatetotal(salt string) int {
	total := 0
	for _, r := range salt {
		total += int(r)
	}
	return total
}

func reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

func Splititem(item string) (channel, key string, data string) {
	splitted := strings.Split(item, ".")

	if len(splitted) > 0 {
		channel = splitted[0]
	}

	if len(splitted) > 1 {
		key = splitted[1]
	}

	if len(splitted) > 2 {
		data = splitted[2]
	}

	if len(channel) == 0 || len(key) == 0 {
		return "", "", ""
	}

	return channel, key, data
}
