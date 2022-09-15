parse<-function (xml){
  require(XML)
  require(dplyr)
  d<-XML::xmlTreeParse(xml, useInternalNodes = T)
  barcode <- xpathSApply(d,path = "//InspectionDetailsItem[Name='Bar Code']//Details",xmlValue)
  tbl <- xpathSApply(d, path = "//List//InspectionDetailsItem[Name='Results']//Details",xmlValue)
  html_tree <- XML::htmlTreeParse(tbl, useInternalNodes = T)
  tds <- xpathApply(html_tree, path = "//td")
  strs <- xmlApply(tds, getChildrenStrings, len = 60)
  dfs <-lapply(strs, pull_cells) %>%  dplyr::bind_rows()
  dfmeta<-data.frame(
    Lot=paste0(substr(barcode, 1, 1), substr(barcode,7, 11)),
    sn = substr(barcode, 2, 6),
    type =  (substr(barcode, 1, 1))
  )
  list(dfmeta,dfs) %>%
  {suppressMessages(dplyr::bind_cols(.))} %>%
  tibble::remove_rownames(.) %>%
    arrange(Well)
}