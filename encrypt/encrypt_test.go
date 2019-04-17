package encrypt

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAesCBCEncrypt(t *testing.T) {
	rawData := []byte(`aaaaaa`)
	key := []byte(`hg62159393`)
	enc, err := Encrypt(rawData, key)
	assert.NoError(t, err)
	assert.Equal(t, "HaxDzm34lg0MFEsI+xBwFg==", enc)
}

func TestAesCBCDncrypt(t *testing.T) {
	rawData := `HaxDzm34lg0MFEsI+xBwFg==`
	key := []byte(`hg62159393`)
	//key := []byte(`0123456789abcdef`)
	enc, err := Dncrypt(rawData, key)
	assert.NoError(t, err)
	//assert.Equal(t, "UubM9ajIurxg8MJtdLi4oQ==", enc)
	assert.Equal(t, "aaaaaa", enc)
}

func TestAppendCopy(t *testing.T) {
	srcSlice := []int{1, 2, 3}
	destSlice := []int{5, 5, 5, 5, 5, 5, 5}
	copy(srcSlice[3:], destSlice)
	fmt.Println("srcSlice = ", srcSlice)
}

func TestAESBase64Encrypt(t *testing.T) {
	rawData := (`12345`)
	key := (`hg62159393`)

	enc, err := AESBase64Encrypt(rawData, key)
	assert.NoError(t, err)
	assert.Equal(t, "NK70GParXbt2OezynLUSPA==", enc)

	rawData = `aaaa`
	enc, err = AESBase64Encrypt(rawData, key)
	assert.NoError(t, err)
	assert.Equal(t, "pt88b59mPhQElLWbJZIHFw==", enc)

	rawData = `qazwsxc12`
	enc, err = AESBase64Encrypt(rawData, key)
	assert.NoError(t, err)
	assert.Equal(t, "mKY3QO3gC76DncjfZJrFeA==", enc)

	//rawData = `20190409135500123_7777`
	//enc, err = AESBase64Encrypt(rawData,key)
	//assert.NoError(t, err)
	//assert.Equal(t, "DoHbro2kzZAJYLObTMGXvSMiVUiOtvOiOxTTWdVMts8=", enc)

	//java := map[string]string{"args":"fbIs5g908HJTWCdc8V3dYA==","enterpass":"YrA4Ik/frKWk2zw64GB0Dw==","caller":"4aFb1PElPqYrFplSCol6LA==","msgType":"32qnXo6OwhUAvbQgtKu5Qg==","mobile":"TZ05Z7T7hO05JDzNe7T9sw==","operid":"xirWM1pIOXs8nWxm5dFAQA==","tempid":"yK1uxETbEFv8T0ILpXjZHA==","sequenceid":"DoHbro2kzZAJYLObTMGXvSMiVUiOtvOiOxTTWdVMts8=","enterid":"llzIbBP7I7PNl4d9EQ9+jA=="}
	//m := map[string]string{"args":"ClientName","enterpass":"ZTTH008","caller":"01057624343","msgType":"4","mobile":"18627826073","operid":"7777","tempid":"5050408","sequenceid":"20190409135500123_7777","enterid":"ZTTHSX20190402"}
	java := map[string]string{"args": "fbIs5g908HJTWCdc8V3dYA==", "enterpass": "YrA4Ik/frKWk2zw64GB0Dw==", "caller": "4aFb1PElPqYrFplSCol6LA==", "msgType": "32qnXo6OwhUAvbQgtKu5Qg==", "mobile": "TZ05Z7T7hO05JDzNe7T9sw==", "operid": "xirWM1pIOXs8nWxm5dFAQA==", "tempid": "yK1uxETbEFv8T0ILpXjZHA==", "enterid": "llzIbBP7I7PNl4d9EQ9+jA=="}
	m := map[string]string{"args": "ClientName", "enterpass": "ZTTH008", "caller": "01057624343", "msgType": "4", "mobile": "18627826073", "operid": "7777", "tempid": "5050408", "enterid": "ZTTHSX20190402"}
	for k, v := range m {
		enc, err = AESBase64Encrypt(v, key)
		assert.NoError(t, err)
		javaValue := java[k]
		assert.Equal(t, javaValue, enc)
	}

}
