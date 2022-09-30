aquire<- function(u){
  require(parallel)
  fls <- locate_details(u)
  n_cores<-detectCores()
  cl <- makeCluster(n_cores)
  df <- parLapply(cl, fls, vis::parse) %>% 
    dplyr::bind_rows()
  parallel::stopCluster(cl)
  df
}
