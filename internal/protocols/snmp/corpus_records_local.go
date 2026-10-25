package snmp
import("bufio";"strings")
// Independent audit assembly: expected literal bytes do not use the production string decoder.
type corpusRecordsLocal struct{scanner *bufio.Scanner;recording bool;text,pending string;have bool}
func(s *corpusRecordsLocal)Scan()bool{var raw string;if s.have{raw=s.pending;s.have=false}else{if !s.scanner.Scan(){return false};raw=s.scanner.Text()};s.text=raw;if s.recording{return true};_,rhs,ok:=strings.Cut(raw,"=");kind,value,colon:=strings.Cut(rhs,":");kind=strings.ToUpper(strings.TrimSpace(kind));value=strings.TrimLeft(value," \t");if !ok||!colon||(kind!="STRING"&&kind!="OCTET STRING")||!strings.HasPrefix(value,"\"")||corpusClosedLocal(value)&&len(value)>1{return true};for s.scanner.Scan(){next:=s.scanner.Text();key,_,binding:=strings.Cut(next,"=");key=strings.TrimSpace(key);if binding&&!strings.ContainsAny(key," \t"){_,numeric:=NormalizeWalkOID(key);if numeric||strings.Contains(key,"::"){s.pending=next;s.have=true;return true}};s.text+="\n"+next;if corpusClosedLocal(next){return true}};return true}
func(s *corpusRecordsLocal)Text()string{return s.text}
func(s *corpusRecordsLocal)Err()error{return s.scanner.Err()}
func corpusClosedLocal(s string)bool{s=strings.TrimRight(s," \t\r");if !strings.HasSuffix(s,"\""){return false};slashes:=0;for i:=len(s)-2;i>=0&&s[i]=='\\';i--{slashes++};return slashes%2==0}
func corpusLiteralStringLocal(record string)(string,bool){_,rhs,ok:=strings.Cut(record,"=");kind,value,colon:=strings.Cut(rhs,":");kind=strings.ToUpper(strings.TrimSpace(kind));value=strings.TrimSpace(value);if !ok||!colon||(kind!="STRING"&&kind!="OCTET STRING"){return "",false};if !strings.HasPrefix(value,"\""){return value,true};if len(value)<2||!corpusClosedLocal(value){return "",false};return value[1:len(value)-1],true}
