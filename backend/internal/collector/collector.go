package collector

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"dc-cooling-optimizer/internal/db"
)

type DeviceConfig struct {
	UnitID     int
	DeviceID   int
	DeviceType string
	BaseAddr   int
	Addr       int
}

type Collector struct {
	db            *db.DB
	modbusAddr    string
	deviceConfigs []DeviceConfig
	stopCh        chan struct{}
	dataCh        chan *db.DeviceData
	DataChannel   func(*db.DeviceData)
	txID          uint16
}

func New(database *db.DB, modbusAddr string) *Collector {
	c := &Collector{
		db:         database,
		modbusAddr: modbusAddr,
		stopCh:     make(chan struct{}),
		dataCh:     make(chan *db.DeviceData, 500),
	}

	for i := 0; i < 8; i++ {
		c.deviceConfigs = append(c.deviceConfigs, DeviceConfig{
			UnitID:     i + 1,
			DeviceID:   i + 1,
			DeviceType: "chiller",
			BaseAddr:   0,
			Addr:       0 + i*20,
		})
	}

	for i := 0; i < 12; i++ {
		c.deviceConfigs = append(c.deviceConfigs, DeviceConfig{
			UnitID:     9 + i,
			DeviceID:   9 + i,
			DeviceType: "cooling_tower",
			BaseAddr:   100,
			Addr:       100 + i*20,
		})
	}

	for i := 0; i < 80; i++ {
		c.deviceConfigs = append(c.deviceConfigs, DeviceConfig{
			UnitID:     21 + i,
			DeviceID:   21 + i,
			DeviceType: "precision_ac",
			BaseAddr:   200,
			Addr:       200 + i*20,
		})
	}

	for i := 0; i < 20; i++ {
		c.deviceConfigs = append(c.deviceConfigs, DeviceConfig{
			UnitID:     101 + i,
			DeviceID:   101 + i,
			DeviceType: "cdu",
			BaseAddr:   400,
			Addr:       400 + i*20,
		})
	}

	return c
}

func (c *Collector) Start(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	c.collectAll(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		case <-ticker.C:
			c.collectAll(ctx)
		}
	}
}

func (c *Collector) Stop() {
	close(c.stopCh)
}

func (c *Collector) collectAll(ctx context.Context) {
	for _, cfg := range c.deviceConfigs {
		data, err := c.collectDevice(ctx, cfg)
		if err != nil {
			continue
		}

		select {
		case c.dataCh <- data:
		default:
		}

		if c.DataChannel != nil {
			c.DataChannel(data)
		}
	}
}

func (c *Collector) readModbus(unitID int, startAddr int, count int) ([]uint16, error) {
	conn, err := net.DialTimeout("tcp", c.modbusAddr, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(5 * time.Second))

	c.txID++
	req := make([]byte, 12)
	binary.BigEndian.PutUint16(req[0:2], c.txID)
	binary.BigEndian.PutUint16(req[2:4], 0)
	binary.BigEndian.PutUint16(req[4:6], 6)
	req[6] = byte(unitID)
	req[7] = 0x03
	binary.BigEndian.PutUint16(req[8:10], uint16(startAddr))
	binary.BigEndian.PutUint16(req[10:12], uint16(count))

	if _, err := conn.Write(req); err != nil {
		return nil, fmt.Errorf("write: %w", err)
	}

	header := make([]byte, 7)
	if _, err := conn.Read(header); err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}

	respLen := int(binary.BigEndian.Uint16(header[4:6]))
	pdu := make([]byte, respLen-1)
	if _, err := conn.Read(pdu); err != nil {
		return nil, fmt.Errorf("read pdu: %w", err)
	}

	if pdu[0] != byte(unitID) {
		return nil, fmt.Errorf("unit id mismatch: got %d, want %d", pdu[0], unitID)
	}

	if pdu[1] != 0x03 {
		if pdu[1]&0x80 != 0 {
			return nil, fmt.Errorf("modbus exception: code %d", pdu[2])
		}
		return nil, fmt.Errorf("function code mismatch: got %d", pdu[1])
	}

	byteCount := int(pdu[2])
	if byteCount != count*2 {
		return nil, fmt.Errorf("byte count mismatch: got %d, want %d", byteCount, count*2)
	}

	registers := make([]uint16, count)
	for i := 0; i < count; i++ {
		registers[i] = binary.BigEndian.Uint16(pdu[3+i*2 : 5+i*2])
	}

	return registers, nil
}

func (c *Collector) collectDevice(ctx context.Context, cfg DeviceConfig) (*db.DeviceData, error) {
	registers, err := c.readModbus(cfg.UnitID, cfg.Addr, 14)
	if err != nil {
		return nil, fmt.Errorf("read modbus unit %d: %w", cfg.UnitID, err)
	}

	data := &db.DeviceData{
		Time:            time.Now(),
		DeviceID:        cfg.DeviceID,
		SupplyTemp:      float64(registers[0]) / 10.0,
		ReturnTemp:      float64(registers[2]) / 10.0,
		FlowRate:        float64(registers[4]) / 10.0,
		Power:           float64(registers[6]) / 10.0,
		Pressure:        float64(registers[8]) / 10.0,
		COP:             float64(registers[10]) / 10.0,
		CoolingCapacity: float64(registers[12]) / 10.0,
		Status:          1,
	}

	if err := c.db.InsertDeviceData(ctx, data); err != nil {
		return nil, fmt.Errorf("insert device data: %w", err)
	}

	return data, nil
}
