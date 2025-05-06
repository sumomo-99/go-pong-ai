package main

import (
	"encoding/json"
	"fmt"
	"image/color"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	screenWidth = 640
	screenHight = 480
	paddleWidth = 20
	paddleHeight = 80
	ballRadius = 10
	paddleSpeed = 50
	ballSpeedX = 50
	ballSpeedY = 50

	ballXDivisions = 3
	ballYDivisions = 3
	paddleYDivisions = 3

	learningRate = 0.1
	discountRate = 0.9
	initialEpsilon = 0.1
	epsilonDecayRate = 0.001
	minEpsilon = 0.01

	velXRight = 1
	velXLeft = -1
	velYUp = 1
	velYDown = -1
)

type Game struct {
	paddle1Y float64
	paddle2Y float64
	ballX float64
	ballY float64
	ballVelX float64
	ballVelY float64
	agent1 *Agent
	agent2 *Agent
	state1 int
	state2 int
	score1 int
	score2 int
	prevState1 int
	prevState2 int
	prevAction1 int
	prevAction2 int
	episodeCount int
}

type Agent struct {
	paddleID int
	qTable map[int]map[int]float64
	learningRate float64
	discountRate float64
	epsilon float64
}

func NewAgent(id int, learningRate, discountRate, epsilon float64) *Agent {
	return &Agent {
		paddleID: id,
		qTable: make(map[int]map[int]float64),
		learningRate: learningRate,
		discountRate: discountRate,
		epsilon: epsilon,
	}
}

func (a *Agent) initializeQTable(numStates, numActions int) {
	a.qTable = make(map[int]map[int]float64)
	for s := 0; s < numStates; s++ {
		a.qTable[s] = make(map[int]float64)
		for act := 0; act < numActions; act++ {
			a.qTable[s][act] = 0.0
		}
	}
}

func (a *Agent) getQValue(state int, action int) float64 {
	if _, ok := a.qTable[state]; !ok {
		return 0.0
	}
	return a.qTable[state][action]
}

func (a *Agent) setQValue(state int, action int, value float64) {
	if _, ok := a.qTable[state]; !ok {
		a.qTable[state] = make(map[int]float64)
	}
	a.qTable[state][action] = value
}

const (
	ActionUp = 0
	ActionDown = 1
	ActionStay = 2
)

func (g *Game) getState(paddleID int) int {
	var ballXDiscrete int
	if g.ballX < screenWidth/3 {
		ballXDiscrete = 0
	} else if g.ballX < 2*screenWidth/3 {
		ballXDiscrete = 1
	} else {
		ballXDiscrete = 2
	}

	var ballYDiscrete int
	if g.ballY < screenHight/3 {
		ballYDiscrete = 0
	} else if g.ballY < 2*screenHight/3 {
		ballYDiscrete = 1
	} else {
		ballYDiscrete = 2
	}

	var paddleYDiscrete int
	var paddleY float64
	if paddleID == 1 {
		paddleY = g.paddle1Y + paddleHeight/2
	} else {
		paddleY = g.paddle2Y + paddleHeight/2
	}
	if paddleY < screenHight/3 {
		paddleYDiscrete = 0
	} else if paddleY < 2*screenHight/3 {
		paddleYDiscrete = 1
	} else {
		paddleYDiscrete = 2
	}

	ballVelXDir := 0
	if g.ballVelX > 0 {
		ballVelXDir = velXRight
	} else {
		ballVelXDir = velXLeft
	}

	ballVelYDir := 0
	if g.ballVelY > 0 {
		ballVelYDir = velYUp
	} else {
		ballVelYDir = velYDown
	}

	var velXState int
	if ballVelXDir > 0 {
		velXState = 0
	} else {
		velXState = 1
	}

	var velYState int
	if ballVelYDir > 0 {
		velYState = 0
	} else {
		velYState = 1
	}

	var realteivePaddleYDiscrete int
	var paddleYCenter float64
	if paddleID == 1 {
		paddleYCenter = g.paddle1Y + paddleHeight/2
	} else {
		paddleYCenter = g.paddle2Y + paddleHeight/2
	}
	relativeY := g.ballY - paddleYCenter
	if relativeY < -paddleHeight/2 {
		realteivePaddleYDiscrete = 0
	} else if relativeY > paddleHeight/2 {
		realteivePaddleYDiscrete = 2
	} else {
		realteivePaddleYDiscrete = 1
	}

	stateID := ballXDiscrete + ballYDiscrete*ballXDivisions +
	  paddleYDiscrete*ballXDivisions*ballYDivisions +
	  velXState*ballXDivisions*ballYDivisions*paddleYDivisions +
	  velYState*ballXDivisions*ballYDivisions*paddleYDivisions*2 +
	  realteivePaddleYDiscrete*ballXDivisions*ballYDivisions*paddleYDivisions*2*3

	return stateID
}

func NewGame() *Game {
	agent1 := NewAgent(1, learningRate, discountRate, initialEpsilon)
	agent2 := NewAgent(2, learningRate, discountRate, initialEpsilon)
	numStates := ballXDivisions * ballYDivisions * paddleYDivisions * 2 * 2 * 3
	agent1.initializeQTable(numStates, 3)
	agent2.initializeQTable(numStates, 3)

	return &Game{
		paddle1Y: float64(screenHight/2 - paddleHeight/2),
		paddle2Y: float64(screenHight/2 - paddleHeight/2),
		ballX: float64(screenWidth/2),
		ballY: float64(screenHight/2),
		ballVelX: ballSpeedX,
		ballVelY: ballSpeedY,
		agent1: agent1,
		agent2: agent2,
		score1: 0,
		score2: 0,
		prevState1: 0,
		prevAction1: ActionStay,
		prevState2: 0,
		prevAction2: ActionStay,
		episodeCount: 0,
	}
}

func (g *Game) Update() error {
	currentState1 := g.getState(1)
	currentState2 := g.getState(2)

	action1 := g.agent1.selectAction(currentState1)
	action2 := g.agent2.selectAction(currentState2)

	g.updatePaddlePosition(1, action1)
	g.updatePaddlePosition(2, action2)

	g.ballX += g.ballVelX
	g.ballY += g.ballVelY

	ballMinX := g.ballX - ballRadius
	ballMinY := g.ballY - ballRadius
	ballMaxX := g.ballX + ballRadius
	ballMaxY := g.ballY + ballRadius

	if ballMinY < 0 || ballMaxY > screenHight {
		g.ballVelY *= -1
	}

	paddle1MinX := float64(50)
	paddle1MinY := g.paddle1Y
	paddle1MaxX := paddle1MinX + paddleWidth
	paddle1MaxY := paddle1MinY + paddleHeight

	paddle2MinX := float64(screenWidth - 50 - paddleWidth)
	paddle2MinY := g.paddle2Y
	paddle2MaxX := paddle2MinX + paddleWidth
	paddle2MaxY := paddle2MinY + paddleHeight

	reward1 := 0.0
	reward2 := 0.0

	if intersect(paddle1MinX, paddle1MinY, paddle1MaxX, paddle1MaxY, ballMinX, ballMinY, ballMaxX, ballMaxY) {
		  g.ballVelX *= -1
		  reward1 += 0.1
		  reward2 -= 0.01
	}

	if intersect(paddle2MinX, paddle2MinY, paddle2MaxX, paddle2MaxY, ballMinX, ballMinY, ballMaxX, ballMaxY) {
		  g.ballVelX *= -1
		  reward2 += 0.1
		  reward1 -= 0.01
	}

	if ballMinX < 0 {
		g.score2++
		reward2 += 1
		reward1 -= 1
		g.resetBall()
	}
	if ballMinX > screenWidth {
		g.score1++
		reward1 += 1
		reward2 -= 1
		g.resetBall()
	}

	g.agent1.updateQValue(g.prevState1, g.prevAction1, reward1, currentState1)
	g.agent2.updateQValue(g.prevState2, g.prevAction2, reward2, currentState2)

	g.prevState1 = currentState1
	g.prevAction1 = action1
	g.prevState2 = currentState2
	g.prevAction2 = action2

	return nil
}

func (g *Game) resetBall() {
	g.episodeCount++
	if g.episodeCount%100 == 0 && g.agent1.epsilon > minEpsilon && g.agent2.epsilon > minEpsilon {
		g.agent1.epsilon -= epsilonDecayRate
		g.agent2.epsilon -= epsilonDecayRate
	}
	g.ballX = float64(screenWidth/2)
	g.ballY = float64(screenHight/2)
	g.ballVelX *= -1
	g.ballVelY = ballSpeedY * (rand.Float64()*2 - 1)
}

func intersect(r1MinX, r1MinY, r1MaxX, r1MaxY, r2MinX, r2MinY, r2MaxX, r2MaxY float64) bool {
	return r1MinX < r2MaxX && r1MaxX > r2MinX && r1MinY < r2MaxY && r1MaxY > r2MinY
}

func (g *Game) updatePaddlePosition(paddleID int, action int) {
	var speed float64 = paddleSpeed

	switch action {
	case ActionUp:
		if paddleID == 1 {
			g.paddle1Y -= speed
			if g.paddle1Y < 0 {
				g.paddle1Y = 0
			}
		} else if paddleID == 2 {
			g.paddle2Y -= speed
			if g.paddle2Y < 0 {
				g.paddle2Y = 0
			}
		}
	case ActionDown:
		if paddleID == 1 {
			g.paddle1Y += speed
			if g.paddle1Y > screenHight-paddleHeight {
				g.paddle1Y = screenHight - paddleHeight
			}
		} else if paddleID == 2 {
			g.paddle2Y += speed
			if g.paddle2Y > screenHight - paddleHeight {
				g.paddle2Y = screenHight - paddleHeight
			}
		}
	case ActionStay:

	}
}

func (a *Agent) selectAction(state int) int {
	if rand.Float64() < a.epsilon {
		return rand.Intn(3)
	} else {
		bestAction := -1
		maxQvalue := -1.0
		if _, ok := a.qTable[state]; ok {
			for action := 0; action < 3; action++ {
				qValue := a.getQValue(state, action)
				if qValue > maxQvalue {
					maxQvalue = qValue
					bestAction = action
				}
			}
		}
		if bestAction == -1 {
			return rand.Intn(3)
		}
		return bestAction
	}
}

func (a *Agent) updateQValue(currentState int, action int, reward float64, nextState int) {
	maxNextQ := 0.0
	if _, ok := a.qTable[nextState]; ok {
		for action := 0; action < 3; action++ {
			qValue := a.getQValue(nextState, action)
			if qValue > maxNextQ {
				maxNextQ  = qValue
			}
		}
	}

	oldQValue := a.getQValue(currentState, action)
	newQValue := oldQValue + a.learningRate*(reward+a.discountRate*maxNextQ-oldQValue)
	a.setQValue(currentState, action, newQValue)
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0, 0, 0, 0xff})
	ebitenutil.DrawRect(screen, 50, g.paddle1Y, paddleWidth, paddleHeight, color.White)
	ebitenutil.DrawRect(screen, screenWidth-50-paddleWidth, g.paddle2Y, paddleWidth, paddleHeight, color.White)
	ebitenutil.DrawCircle(screen, g.ballX, g.ballY, ballRadius, color.White)

	scoreText := fmt.Sprintf("AI 1: %d  AI 2: %d", g.score1, g.score2)
	ebitenutil.DebugPrint(screen, scoreText)

	episodeText := fmt.Sprintf("Episode: %d", g.episodeCount)
	ebitenutil.DebugPrintAt(screen, episodeText, 0, 20)

	epsilonText1 := fmt.Sprintf("Epsilon 1: %.2f", g.agent1.epsilon)
	ebitenutil.DebugPrintAt(screen, epsilonText1, 0, 40)

	epsilonText2 := fmt.Sprintf("Epsilon 2: %.2f", g.agent2.epsilon)
	ebitenutil.DebugPrintAt(screen, epsilonText2, 0, 60)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHight
}

func (a *Agent) SaveQTable(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(a.qTable); err != nil {
		return fmt.Errorf("failed to encode Q-table: %w", err)
	}
	fmt.Printf("Q-table for agent %d saved to %s\n", a.paddleID, filename)

	return nil
}

func (a *Agent) LoadQTable(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("Q-table file %s not found for agent %d. Starting with an empty Q-table.\n", filename, a.paddleID)
			return nil
		}
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&a.qTable); err != nil {
		return fmt.Errorf("failed to decode Q-table: %w", err)
	}
	fmt.Printf("Q-table for agent %d loaded from %s\n", a.paddleID, filename)
	return nil
}

func main() {
	rand.Seed(time.Now().UnixNano())

	game := NewGame()

	err1 := game.agent1.LoadQTable("agent1_q_table.json")
	if err1 != nil {
		log.Println("Error loading Q-table for agent 1: %w", err1)
	}
	err2 := game.agent2.LoadQTable("agent2_q_table.json")
	if err2 != nil {
		log.Println("Error loading Q-table for agent 2: %w", err2)
	}

	ebiten.SetWindowSize(screenWidth, screenHight)
	ebiten.SetWindowTitle("Pong AI")

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}

	errSave1 := game.agent1.SaveQTable("agent1_q_table.json")
	if errSave1 != nil {
		log.Println("Error saving Q-table for agent 1: %w", errSave1)
	}
	errSave2 := game.agent2.SaveQTable("agent2_q_table.json")
	if errSave2 != nil {
		log.Println("Error saving Q-table for agent 2: %w", errSave2)
	}
}
