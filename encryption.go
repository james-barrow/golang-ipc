package ipc

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"io"
	"log"
	"net"
)

//Enc -
func Enc() {

	//priva, _ := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	//puba := priva.PublicKey

	//privb, _ := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	//pubb := privb.PublicKey

	//fmt.Printf("\nPrivate key (Alice) %x", priva.D)
	//fmt.Printf("\nPublic key (Alice) (%x,%x)", puba.X, puba.Y)

	//fmt.Printf("\nPrivate key (Bob) %x", privb.D)
	//fmt.Printf("\nPublic key (Bob) (%x,%x)\n", pubb.X, pubb.Y)

	//fmt.Sprintf("%x", pubb.X)

	//fmt.Println(len(fmt.Sprintf("%x", pubb.X)))

	//a, _ := puba.Curve.ScalarMult(puba.X, puba.Y, privb.D.Bytes())

	//b, _ := pubb.Curve.ScalarMult(pubb.X, pubb.Y, priva.D.Bytes())

	//shared1 := sha256.Sum256(a.Bytes())
	//shared2 := sha256.Sum256(b.Bytes())

	//fmt.Printf("\nShared key (Alice) %x", shared1)
	///fmt.Printf("\nShared key  (Bob)  %x\n", shared2)

	_, pub, err := generateKeys()
	if err != nil {
		log.Println("KEY GEN ERROR : " + err.Error())
	}

	pubSend := publicKeyToBytes(pub)

	recvdPub := bytesToPublicKey(pubSend)

	if recvdPub == nil {
		log.Println("got nil bytes : ")
	}

	if recvdPub.IsOnCurve(recvdPub.X, recvdPub.Y) == false {
		log.Println("Cleint KEY not on curve")
	} else {
		log.Println("goodGOOOOooD")
	}

}

func (sc *Server) keyExchange() ([32]byte, error) {

	var shared [32]byte

	priv, pub, err := generateKeys()
	if err != nil {
		return shared, err
	}

	// send servers public key
	err = sendPublic(sc.conn, pub)
	if err != nil {
		return shared, err
	}

	// recieve clients public key
	pubRecvd, err := recvPublic(sc.conn)
	if err != nil {
		return shared, err
	}

	b, _ := pubRecvd.Curve.ScalarMult(pubRecvd.X, pubRecvd.Y, priv.D.Bytes())

	shared = sha256.Sum256(b.Bytes())

	return shared, nil

}

func (cc *Client) keyExchange() ([32]byte, error) {

	var shared [32]byte

	priv, pub, err := generateKeys()
	if err != nil {
		return shared, err
	}

	// recieve servers public key
	pubRecvd, err := recvPublic(cc.conn)
	if err != nil {
		return shared, err
	}

	// send clients public key
	err = sendPublic(cc.conn, pub)
	if err != nil {
		return shared, err
	}

	b, _ := pubRecvd.Curve.ScalarMult(pubRecvd.X, pubRecvd.Y, priv.D.Bytes())

	shared = sha256.Sum256(b.Bytes())

	return shared, nil
}

func generateKeys() (*ecdsa.PrivateKey, *ecdsa.PublicKey, error) {

	priva, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	puba := &priva.PublicKey

	if priva.IsOnCurve(puba.X, puba.Y) == false {
		return nil, nil, errors.New("keys created arn't on curve")
	}

	return priva, puba, err

}

func sendPublic(conn net.Conn, pub *ecdsa.PublicKey) error {

	pubSend := publicKeyToBytes(pub)
	if pubSend == nil {
		return errors.New("public key cannot be converted to bytes")
	}

	_, err := conn.Write(pubSend)
	if err != nil {
		return errors.New("could not sent public key")
	}

	return nil
}

func recvPublic(conn net.Conn) (*ecdsa.PublicKey, error) {

	buff := make([]byte, 300)
	i, err := conn.Read(buff)
	if err != nil {
		return nil, errors.New("didn't recieve public key")
	}

	recvdPub := bytesToPublicKey(buff[:i])

	if recvdPub.IsOnCurve(recvdPub.X, recvdPub.Y) == false {
		return nil, errors.New("didn't recieve valid public key")
	}

	return recvdPub, nil
}

func publicKeyToBytes(pub *ecdsa.PublicKey) []byte {

	if pub == nil || pub.X == nil || pub.Y == nil {
		return nil
	}

	return elliptic.Marshal(elliptic.P384(), pub.X, pub.Y)
}

func bytesToPublicKey(recvdPub []byte) *ecdsa.PublicKey {

	if len(recvdPub) == 0 {
		return nil
	}

	x, y := elliptic.Unmarshal(elliptic.P384(), recvdPub)
	return &ecdsa.PublicKey{Curve: elliptic.P384(), X: x, Y: y}

}

/*
func hexToPublicKey(xHex string, yHex string) *ecdsa.PublicKey {
	xBytes, _ := hex.DecodeString(xHex)
	x := new(big.Int)
	x.SetBytes(xBytes)

	yBytes, _ := hex.DecodeString(yHex)
	y := new(big.Int)
	y.SetBytes(yBytes)

	pub := new(ecdsa.PublicKey)
	pub.X = x
	pub.Y = y

	pub.Curve = elliptic.P384()

	return pub
}
*/

func createCipher(shared [32]byte) (*cipher.AEAD, error) {

	b, err := aes.NewCipher(shared[:])

	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(b)
	if err != nil {
		return nil, err
	}

	return &gcm, nil
}

func encrypt(g cipher.AEAD, data []byte) ([]byte, error) {

	nonce := make([]byte, g.NonceSize())

	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return g.Seal(nonce, nonce, data, nil), nil

}

func decrypt(g cipher.AEAD, recdData []byte) ([]byte, error) {

	nonceSize := g.NonceSize()
	if len(recdData) < nonceSize {
		return nil, errors.New("not enough data to decrypt")

	}

	nonce, recdData := recdData[:nonceSize], recdData[nonceSize:]
	plain, err := g.Open(nil, nonce, recdData, nil)
	if err != nil {
		return nil, err
	}

	return plain, nil

}
