package jwz

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/iden3/go-circuits"
	circuitsTesting "github.com/iden3/go-circuits/testing"
	"github.com/iden3/go-schema-processor/verifiable"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

func TestNewWithPayload(t *testing.T) {
	payload := []byte("mymessage")
	token, err := NewWithPayload(ProvingMethodGroth16AuthInstance, payload)
	assert.NoError(t, err)

	assert.Equal(t, "groth16", token.Alg)
	assert.Equal(t, "auth", token.CircuitID)
	assert.Equal(t, []HeaderKey{headerCircuitID}, token.raw.Header[headerCritical])
	assert.Equal(t, "groth16", token.raw.Header[headerAlg])
}

func TestToken_Prove(t *testing.T) {
	payload := []byte("mymessage")
	token, err := NewWithPayload(ProvingMethodGroth16AuthInstance, payload)
	assert.NoError(t, err)

	msgHash, err := token.GetMessageHash()
	assert.NoError(t, err)
	challenge := new(big.Int).SetBytes(msgHash)

	var provingkey interface{}

	ctx := context.Background()
	privKeyHex := "28156abe7fe2fd433dc9df969286b96666489bac508612d0e16593e944c4f69f"
	identifier, claim, state, claimsTree, revTree, rootsTree, claimEntryMTP, claimNonRevMTP, signature, err := circuitsTesting.AuthClaimFullInfo(ctx, privKeyHex, challenge)
	assert.Nil(t, err)

	treeState := circuits.TreeState{
		State:          state,
		ClaimsRoot:     claimsTree.Root(),
		RevocationRoot: revTree.Root(),
		RootOfRoots:    rootsTree.Root(),
	}

	inputs := circuits.AuthInputs{
		ID: identifier,
		AuthClaim: circuits.Claim{
			Claim:       claim,
			Proof:       claimEntryMTP,
			TreeState:   treeState,
			NonRevProof: circuits.ClaimNonRevStatus{TreeState: treeState, Proof: claimNonRevMTP},
		},
		Signature: signature,
		Challenge: challenge,
	}

	err = token.Prove(inputs, provingkey)

	assert.NoError(t, err)

	tokenString, err := token.CompactSerialize()
	assert.NoError(t, err)
	t.Log(tokenString)

}

func TestToken_Parse(t *testing.T) {

	token, err := Parse("eyJhbGciOiJncm90aDE2IiwiY2lyY3VpdElkIjoiYXV0aCIsImNyaXQiOlsiY2lyY3VpdElkIl0sInR5cCI6IkpXWiJ9.bXltZXNzYWdl.eyJwcm9vZiI6eyJwaV9hIjpbIjEzNTI4OTkwMDk0MDQxMTMzNzcwOTg3Njg3NzUzNzUxNjMzMTU4OTUwMTYwMTIwMjgzNTU0ODI0ODUwMzE4MDE4NTExNDYyMzI1NTciLCI3ODgwNDc1MzY2MjU3ODA4ODUzMTM1NDg4MDUwOTkyNTEyMzE3NzA3OTU2ODA3NTA0NzM2NTkwMzAwMTM0Njg3NTMzMjM4MDU4MTU3IiwiMSJdLCJwaV9iIjpbWyIxNzk0NTcxMzI1ODk1OTQ0OTIyMjk0NzUzMTIxNDQyOTk3ODY5NjIxMzg5NjEzNTU2MzAwNjIxOTgwNzg5MDg5NTU2MTE1MzE1Mjc2MiIsIjEzNDMwMzU3MDgyODc5Mjc0ODkzNTQ0MDI2NzU4MTkyNzU5NjUzMTkxOTU3NjI0MjkzOTMzMTAwMDY1NDcyMDgxMTcyNjY2NzA4MTUzIl0sWyIyMTU1NTEzMjkyMDk5MDUyMzMwMTYwNjM5ODQxMjMxNDYzMDI0MDAzNDM2NTAwODYxMjQwNzQ0MTU2MTMyMzA1MzYxNjA1MjcyMzA1IiwiMTAzNzYwMTMwMjA1ODIyMzQyOTMzNzE4MDc2NzU0MDg5OTcyNTk0ODczNjE5MzQ4OTY3ODYyNTQ0NzI3MjQ5MDk1NDI0NjYwMzA0NzgiXSxbIjEiLCIwIl1dLCJwaV9jIjpbIjE4ODU1ODYxNzExMzMzNTUxOTgwMzAyNDk5ODg3NDg1MjUxNTU0NDc0NzI3OTQ4OTE4NzEzMDQwNTgzMjA1MjM1NjE3NTA5MTMyMzE5IiwiMTk3MjE5OTMwMjA0ODQzMDk1NDE5MzA2OTU2MTE3MDAwMTc4ODYyOTg2MjY4MjgwMDIyMTMyNDUwNzk4NzU4OTg1MTE1MDI2NzgxNzciLCIxIl0sInByb3RvY29sIjoiZ3JvdGgxNiJ9LCJwdWJfc2lnbmFscyI6WyIxOTA1NDMzMzk3MDg4NTAyMzc4MDEyMzU2MDkzNjY3NTQ1NjcwMDg2MTQ2OTA2ODYwMzMyMTg4NDcxODc0ODk2MTc1MDkzMDQ2Njc5NCIsIjE4NjU2MTQ3NTQ2NjY2OTQ0NDg0NDUzODk5MjQxOTE2NDY5NTQ0MDkwMjU4ODEwMTkyODAzOTQ5NTIyNzk0NDkwNDkzMjcxMDA1MzEzIiwiMzc5OTQ5MTUwMTMwMjE0NzIzNDIwNTg5NjEwOTExMTYxODk1NDk1NjQ3Nzg5MDA2NjQ5Nzg1MjY0NzM4MTQxMjk5MTM1NDE0MjcyIl19")
	assert.NoError(t, err)

	var zkProof verifiable.ZKProof
	proofBytes, err := base64.StdEncoding.DecodeString("eyJwcm9vZiI6eyJwaV9hIjpbIjEzNTI4OTkwMDk0MDQxMTMzNzcwOTg3Njg3NzUzNzUxNjMzMTU4OTUwMTYwMTIwMjgzNTU0ODI0ODUwMzE4MDE4NTExNDYyMzI1NTciLCI3ODgwNDc1MzY2MjU3ODA4ODUzMTM1NDg4MDUwOTkyNTEyMzE3NzA3OTU2ODA3NTA0NzM2NTkwMzAwMTM0Njg3NTMzMjM4MDU4MTU3IiwiMSJdLCJwaV9iIjpbWyIxNzk0NTcxMzI1ODk1OTQ0OTIyMjk0NzUzMTIxNDQyOTk3ODY5NjIxMzg5NjEzNTU2MzAwNjIxOTgwNzg5MDg5NTU2MTE1MzE1Mjc2MiIsIjEzNDMwMzU3MDgyODc5Mjc0ODkzNTQ0MDI2NzU4MTkyNzU5NjUzMTkxOTU3NjI0MjkzOTMzMTAwMDY1NDcyMDgxMTcyNjY2NzA4MTUzIl0sWyIyMTU1NTEzMjkyMDk5MDUyMzMwMTYwNjM5ODQxMjMxNDYzMDI0MDAzNDM2NTAwODYxMjQwNzQ0MTU2MTMyMzA1MzYxNjA1MjcyMzA1IiwiMTAzNzYwMTMwMjA1ODIyMzQyOTMzNzE4MDc2NzU0MDg5OTcyNTk0ODczNjE5MzQ4OTY3ODYyNTQ0NzI3MjQ5MDk1NDI0NjYwMzA0NzgiXSxbIjEiLCIwIl1dLCJwaV9jIjpbIjE4ODU1ODYxNzExMzMzNTUxOTgwMzAyNDk5ODg3NDg1MjUxNTU0NDc0NzI3OTQ4OTE4NzEzMDQwNTgzMjA1MjM1NjE3NTA5MTMyMzE5IiwiMTk3MjE5OTMwMjA0ODQzMDk1NDE5MzA2OTU2MTE3MDAwMTc4ODYyOTg2MjY4MjgwMDIyMTMyNDUwNzk4NzU4OTg1MTE1MDI2NzgxNzciLCIxIl0sInByb3RvY29sIjoiZ3JvdGgxNiJ9LCJwdWJfc2lnbmFscyI6WyIxOTA1NDMzMzk3MDg4NTAyMzc4MDEyMzU2MDkzNjY3NTQ1NjcwMDg2MTQ2OTA2ODYwMzMyMTg4NDcxODc0ODk2MTc1MDkzMDQ2Njc5NCIsIjE4NjU2MTQ3NTQ2NjY2OTQ0NDg0NDUzODk5MjQxOTE2NDY5NTQ0MDkwMjU4ODEwMTkyODAzOTQ5NTIyNzk0NDkwNDkzMjcxMDA1MzEzIiwiMzc5OTQ5MTUwMTMwMjE0NzIzNDIwNTg5NjEwOTExMTYxODk1NDk1NjQ3Nzg5MDA2NjQ5Nzg1MjY0NzM4MTQxMjk5MTM1NDE0MjcyIl19")
	assert.NoError(t, err)
	err = json.Unmarshal(proofBytes, &zkProof)
	assert.NoError(t, err)

	payloadBytes, err := base64.StdEncoding.DecodeString("bXltZXNzYWdl")
	assert.NoError(t, err)

	assert.Equal(t, zkProof.PubSignals, token.ZkProof.PubSignals)
	assert.Equal(t, zkProof.Proof, token.ZkProof.Proof)
	assert.Equal(t, "auth", token.CircuitID)
	assert.Equal(t, "groth16", token.Alg)
	assert.Equal(t, payloadBytes, token.raw.Payload)

}

func TestToken_ParseWithOutputs(t *testing.T) {

	token, err := Parse("eyJhbGciOiJncm90aDE2IiwiY2lyY3VpdElkIjoiYXV0aCIsImNyaXQiOlsiY2lyY3VpdElkIl0sInR5cCI6IkpXWiJ9.bXltZXNzYWdl.eyJwcm9vZiI6eyJwaV9hIjpbIjEzNTI4OTkwMDk0MDQxMTMzNzcwOTg3Njg3NzUzNzUxNjMzMTU4OTUwMTYwMTIwMjgzNTU0ODI0ODUwMzE4MDE4NTExNDYyMzI1NTciLCI3ODgwNDc1MzY2MjU3ODA4ODUzMTM1NDg4MDUwOTkyNTEyMzE3NzA3OTU2ODA3NTA0NzM2NTkwMzAwMTM0Njg3NTMzMjM4MDU4MTU3IiwiMSJdLCJwaV9iIjpbWyIxNzk0NTcxMzI1ODk1OTQ0OTIyMjk0NzUzMTIxNDQyOTk3ODY5NjIxMzg5NjEzNTU2MzAwNjIxOTgwNzg5MDg5NTU2MTE1MzE1Mjc2MiIsIjEzNDMwMzU3MDgyODc5Mjc0ODkzNTQ0MDI2NzU4MTkyNzU5NjUzMTkxOTU3NjI0MjkzOTMzMTAwMDY1NDcyMDgxMTcyNjY2NzA4MTUzIl0sWyIyMTU1NTEzMjkyMDk5MDUyMzMwMTYwNjM5ODQxMjMxNDYzMDI0MDAzNDM2NTAwODYxMjQwNzQ0MTU2MTMyMzA1MzYxNjA1MjcyMzA1IiwiMTAzNzYwMTMwMjA1ODIyMzQyOTMzNzE4MDc2NzU0MDg5OTcyNTk0ODczNjE5MzQ4OTY3ODYyNTQ0NzI3MjQ5MDk1NDI0NjYwMzA0NzgiXSxbIjEiLCIwIl1dLCJwaV9jIjpbIjE4ODU1ODYxNzExMzMzNTUxOTgwMzAyNDk5ODg3NDg1MjUxNTU0NDc0NzI3OTQ4OTE4NzEzMDQwNTgzMjA1MjM1NjE3NTA5MTMyMzE5IiwiMTk3MjE5OTMwMjA0ODQzMDk1NDE5MzA2OTU2MTE3MDAwMTc4ODYyOTg2MjY4MjgwMDIyMTMyNDUwNzk4NzU4OTg1MTE1MDI2NzgxNzciLCIxIl0sInByb3RvY29sIjoiZ3JvdGgxNiJ9LCJwdWJfc2lnbmFscyI6WyIxOTA1NDMzMzk3MDg4NTAyMzc4MDEyMzU2MDkzNjY3NTQ1NjcwMDg2MTQ2OTA2ODYwMzMyMTg4NDcxODc0ODk2MTc1MDkzMDQ2Njc5NCIsIjE4NjU2MTQ3NTQ2NjY2OTQ0NDg0NDUzODk5MjQxOTE2NDY5NTQ0MDkwMjU4ODEwMTkyODAzOTQ5NTIyNzk0NDkwNDkzMjcxMDA1MzEzIiwiMzc5OTQ5MTUwMTMwMjE0NzIzNDIwNTg5NjEwOTExMTYxODk1NDk1NjQ3Nzg5MDA2NjQ5Nzg1MjY0NzM4MTQxMjk5MTM1NDE0MjcyIl19")
	assert.NoError(t, err)

	outs := circuits.AuthPubSignals{}
	err = token.ParsePubSignals(&outs)
	assert.NoError(t, err)

	assert.Equal(t, "119tqceWdRd2F6WnAyVuFQRFjK3WUXq2LorSPyG9LJ", outs.UserID.String())
	assert.Equal(t, "81d8df08abc3e9254b0becbf3d7b01d0f562e417adb4c13d453544485c013f29", outs.UserState.Hex())

	msgHash, err := token.GetMessageHash()
	assert.NoError(t, err)
	assert.Equal(t, msgHash, outs.Challenge.Bytes())
}

func TestToken_Verify(t *testing.T) {

	token, err := Parse("eyJhbGciOiJncm90aDE2IiwiY2lyY3VpdElkIjoiYXV0aCIsImNyaXQiOlsiY2lyY3VpdElkIl0sInR5cCI6IkpXWiJ9.bXltZXNzYWdl.eyJwcm9vZiI6eyJwaV9hIjpbIjEzNTI4OTkwMDk0MDQxMTMzNzcwOTg3Njg3NzUzNzUxNjMzMTU4OTUwMTYwMTIwMjgzNTU0ODI0ODUwMzE4MDE4NTExNDYyMzI1NTciLCI3ODgwNDc1MzY2MjU3ODA4ODUzMTM1NDg4MDUwOTkyNTEyMzE3NzA3OTU2ODA3NTA0NzM2NTkwMzAwMTM0Njg3NTMzMjM4MDU4MTU3IiwiMSJdLCJwaV9iIjpbWyIxNzk0NTcxMzI1ODk1OTQ0OTIyMjk0NzUzMTIxNDQyOTk3ODY5NjIxMzg5NjEzNTU2MzAwNjIxOTgwNzg5MDg5NTU2MTE1MzE1Mjc2MiIsIjEzNDMwMzU3MDgyODc5Mjc0ODkzNTQ0MDI2NzU4MTkyNzU5NjUzMTkxOTU3NjI0MjkzOTMzMTAwMDY1NDcyMDgxMTcyNjY2NzA4MTUzIl0sWyIyMTU1NTEzMjkyMDk5MDUyMzMwMTYwNjM5ODQxMjMxNDYzMDI0MDAzNDM2NTAwODYxMjQwNzQ0MTU2MTMyMzA1MzYxNjA1MjcyMzA1IiwiMTAzNzYwMTMwMjA1ODIyMzQyOTMzNzE4MDc2NzU0MDg5OTcyNTk0ODczNjE5MzQ4OTY3ODYyNTQ0NzI3MjQ5MDk1NDI0NjYwMzA0NzgiXSxbIjEiLCIwIl1dLCJwaV9jIjpbIjE4ODU1ODYxNzExMzMzNTUxOTgwMzAyNDk5ODg3NDg1MjUxNTU0NDc0NzI3OTQ4OTE4NzEzMDQwNTgzMjA1MjM1NjE3NTA5MTMyMzE5IiwiMTk3MjE5OTMwMjA0ODQzMDk1NDE5MzA2OTU2MTE3MDAwMTc4ODYyOTg2MjY4MjgwMDIyMTMyNDUwNzk4NzU4OTg1MTE1MDI2NzgxNzciLCIxIl0sInByb3RvY29sIjoiZ3JvdGgxNiJ9LCJwdWJfc2lnbmFscyI6WyIxOTA1NDMzMzk3MDg4NTAyMzc4MDEyMzU2MDkzNjY3NTQ1NjcwMDg2MTQ2OTA2ODYwMzMyMTg4NDcxODc0ODk2MTc1MDkzMDQ2Njc5NCIsIjE4NjU2MTQ3NTQ2NjY2OTQ0NDg0NDUzODk5MjQxOTE2NDY5NTQ0MDkwMjU4ODEwMTkyODAzOTQ5NTIyNzk0NDkwNDkzMjcxMDA1MzEzIiwiMzc5OTQ5MTUwMTMwMjE0NzIzNDIwNTg5NjEwOTExMTYxODk1NDk1NjQ3Nzg5MDA2NjQ5Nzg1MjY0NzM4MTQxMjk5MTM1NDE0MjcyIl19")
	assert.NoError(t, err)

	err = token.Verify([]byte("{\"protocol\":\"groth16\",\"curve\":\"bn128\",\"nPublic\":3,\"vk_alpha_1\":[\"20491192805390485299153009773594534940189261866228447918068658471970481763042\",\"9383485363053290200918347156157836566562967994039712273449902621266178545958\",\"1\"],\"vk_beta_2\":[[\"6375614351688725206403948262868962793625744043794305715222011528459656738731\",\"4252822878758300859123897981450591353533073413197771768651442665752259397132\"],[\"10505242626370262277552901082094356697409835680220590971873171140371331206856\",\"21847035105528745403288232691147584728191162732299865338377159692350059136679\"],[\"1\",\"0\"]],\"vk_gamma_2\":[[\"10857046999023057135944570762232829481370756359578518086990519993285655852781\",\"11559732032986387107991004021392285783925812861821192530917403151452391805634\"],[\"8495653923123431417604973247489272438418190587263600148770280649306958101930\",\"4082367875863433681332203403145435568316851327593401208105741076214120093531\"],[\"1\",\"0\"]],\"vk_delta_2\":[[\"398933015954148646368938908853947535524860271341483980340526451323240573575\",\"10145678833862082297331325155987602479819448775366292127596703717428972422786\"],[\"10015825487282801267098870968934071805420712655255142726435075910922648491690\",\"16391780571584464089463709222186244276089598803483342252540103888944304766561\"],[\"1\",\"0\"]],\"vk_alphabeta_12\":[[[\"2029413683389138792403550203267699914886160938906632433982220835551125967885\",\"21072700047562757817161031222997517981543347628379360635925549008442030252106\"],[\"5940354580057074848093997050200682056184807770593307860589430076672439820312\",\"12156638873931618554171829126792193045421052652279363021382169897324752428276\"],[\"7898200236362823042373859371574133993780991612861777490112507062703164551277\",\"7074218545237549455313236346927434013100842096812539264420499035217050630853\"]],[[\"7077479683546002997211712695946002074877511277312570035766170199895071832130\",\"10093483419865920389913245021038182291233451549023025229112148274109565435465\"],[\"4595479056700221319381530156280926371456704509942304414423590385166031118820\",\"19831328484489333784475432780421641293929726139240675179672856274388269393268\"],[\"11934129596455521040620786944827826205713621633706285934057045369193958244500\",\"8037395052364110730298837004334506829870972346962140206007064471173334027475\"]]],\"IC\":[[\"448374446865135939647769541890154166042914921041912221286049154597067433244\",\"11506662732119780138569281526472754333062074738857297477066268899841676261677\",\"1\"],[\"14754673471377957198670432699210858035525059415615140923882559652711865422026\",\"21237349496096085376408167399650139181453329473826592789554131167780412319291\",\"1\"],[\"21287922739003385816480150654484934434031086395597647322040273495223392444173\",\"7860334010448425278721389847502162976670934187890845446945915828310145702769\",\"1\"],[\"11065398733589914616819940103547091917179463489998190517391929200753651172846\",\"16010781869086024312453962077351326782445934594671040759005152350849483388443\",\"1\"]]}"))

	assert.NoError(t, err)
	assert.True(t, token.Valid)

}
