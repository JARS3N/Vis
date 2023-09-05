context_serial<-function(doc){
  library(XML)
  library(dplyr)
d <- XML::xmlTreeParse(doc, useInternalNodes = T)
nms<-c("sn","result","rescodes")
paths<-file.path( "//RegressionResult",c("Result","SerialNumber","ResultCodes"))
LoadFilename<-xpathSApply(d,path="//LoadFilename",xmlValue)
cd <- xpathSApply(d, 
      path = paths,
      xmlValue) %>% 
  as.list() %>% 
  setNames(nms) %>% 
  as.data.frame() %>% 
  mutate(Lot=basename(dirname(dirname(LoadFilename))))
cd
}
