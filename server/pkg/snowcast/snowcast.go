package snowcast

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/jennyyu212/cs1680-final-project/pb"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	CHUNK_SIZE     = 1024
	MUSIC_FOLDER   = "./mp3/"
	MUSIC_FILE_EXT = ".mp3"
)

type Connection struct {
	userId string
	stream pb.Snowcast_ConnectServer
	errCh  chan error
}

type SnowcastService struct {
	connections     map[string]*Connection
	connectionsLock sync.RWMutex

	messages    []*pb.Message
	latestIndex int
	msgLock     sync.RWMutex

	pb.UnimplementedSnowcastServer
}

func NewService() *SnowcastService {
	return &SnowcastService{
		connections: make(map[string]*Connection),
		messages:    make([]*pb.Message, 0),
	}
}

func (s *SnowcastService) Connect(request *pb.User, connection pb.Snowcast_ConnectServer) error {
	log.Printf("User %v connected\n", request.GetUserId())

	newConnection := &Connection{
		userId: request.GetUserId(),
		stream: connection,
		errCh:  make(chan error),
	}

	// add to connections
	s.connectionsLock.Lock()
	if _, ok := s.connections[request.GetUserId()]; !ok {
		s.connections[request.UserId] = newConnection
	} else {
		s.connectionsLock.Unlock()
		return fmt.Errorf("user with id %v already exists", request.GetUserId())
	}
	s.connectionsLock.Unlock()

	// notify others
	var wg sync.WaitGroup
	str := new(string)
	*str = fmt.Sprintf("New user connected: %v", newConnection.userId)
	update := &pb.MessageUpdate{
		Announcement: str,
	}
	s.connectionsLock.RLock()
	for _, conn := range s.connections {
		if conn.userId != newConnection.userId {
			wg.Add(1)
			c := conn
			go func() {
				if e := c.stream.Send(update); e != nil {
					c.errCh <- e
				}
				wg.Done()
			}()
		}
	}
	s.connectionsLock.RUnlock()
	wg.Wait()

	e := <-newConnection.errCh
	log.Printf("User %v disconnected\n", request.GetUserId())

	s.connectionsLock.Lock()
	delete(s.connections, request.GetUserId())
	s.connectionsLock.Unlock()

	return e
}

func (s *SnowcastService) GetPlaylist(ctx context.Context, in *emptypb.Empty) (*pb.Playlist, error) {
	dir, err := os.Open(MUSIC_FOLDER)
	if err != nil {
		log.Printf("Error opening music folder: %v\n", err)
		return &pb.Playlist{}, err
	}
	files, err := dir.Readdir(0)
	if err != nil {
		log.Printf("Error reading music folder: %v\n", err)
		return &pb.Playlist{}, err
	}

	playlist := &pb.Playlist{
		Playlist: make([]*pb.Music, 0),
	}

	for _, file := range files {
		filename := strings.TrimSuffix(file.Name(), MUSIC_FILE_EXT)

		playlist.Playlist = append(playlist.Playlist, &pb.Music{
			Name: filename,
		})
	}

	return playlist, nil
}

func (s *SnowcastService) SendMessage(ctx context.Context, message *pb.Message) (*emptypb.Empty, error) {
	log.Printf("User %v sent message: %v\n", message.GetSender(), message.GetMessage())

	s.msgLock.Lock()
	s.messages = append(s.messages, message)
	s.latestIndex++
	s.msgLock.Unlock()

	update := &pb.MessageUpdate{
		LatestMsg: int32(s.latestIndex),
	}

	var wg sync.WaitGroup
	s.connectionsLock.RLock()
	for _, conn := range s.connections {
		wg.Add(1)
		c := conn
		go func() {
			if e := c.stream.Send(update); e != nil {
				c.errCh <- e
			}
			wg.Done()
		}()
	}
	s.connectionsLock.RUnlock()
	wg.Wait()

	log.Printf("Notified all %v users\n", len(s.connections))

	return &emptypb.Empty{}, nil
}

func (s *SnowcastService) FetchMessages(ctx context.Context, request *pb.FetchRequest) (*pb.Messages, error) {
	log.Printf("Fetching messages from %v\n", request.GetStartIndex())

	s.msgLock.RLock()
	defer s.msgLock.RUnlock()

	if request.GetStartIndex() >= int32(len(s.messages)) {
		return nil, fmt.Errorf("start index %v is out of range %v", request.GetStartIndex(), len(s.messages))
	}
	return &pb.Messages{
		Messages: s.messages[request.GetStartIndex():],
	}, nil
}

func (s *SnowcastService) FetchMusic(request *pb.Music, connection pb.Snowcast_FetchMusicServer) error {
	log.Printf("Fetching music %v\n", request.GetName())

	f, e := os.Open(MUSIC_FOLDER + request.GetName() + MUSIC_FILE_EXT)
	if e != nil {
		return e
	}

	chunk := make([]byte, CHUNK_SIZE)
	for {
		n, err := f.Read(chunk)
		if err != nil {
			if err != io.EOF {
				return err
			}
			break
		}

		if err = connection.Send(&pb.FileChunk{
			Chunk: chunk[:n],
		}); err != nil {
			return err
		}

		chunk = make([]byte, CHUNK_SIZE)
	}

	return nil
}
